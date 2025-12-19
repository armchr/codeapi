using CSharpService.Core.Interfaces;
using CSharpService.Core.Models;
using Microsoft.Extensions.Logging;

namespace CSharpService.Core.Services;

/// <summary>
/// Implementation of weather service that coordinates between API, cache, and database
/// </summary>
public class WeatherService : IWeatherService
{
    private readonly IWeatherRepository _repository;
    private readonly IWeatherApiClient _apiClient;
    private readonly ICacheService _cache;
    private readonly ILogger<WeatherService> _logger;
    private readonly TimeSpan _cacheExpiration = TimeSpan.FromMinutes(15);
    private readonly TimeSpan _dataFreshness = TimeSpan.FromMinutes(30);

    public WeatherService(
        IWeatherRepository repository,
        IWeatherApiClient apiClient,
        ICacheService cache,
        ILogger<WeatherService> logger)
    {
        _repository = repository ?? throw new ArgumentNullException(nameof(repository));
        _apiClient = apiClient ?? throw new ArgumentNullException(nameof(apiClient));
        _cache = cache ?? throw new ArgumentNullException(nameof(cache));
        _logger = logger ?? throw new ArgumentNullException(nameof(logger));
    }

    public async Task<ApiResponse<WeatherDto>> GetCurrentWeatherAsync(
        WeatherRequest request,
        CancellationToken cancellationToken = default)
    {
        ArgumentNullException.ThrowIfNull(request);

        var cacheKey = BuildCacheKey(request.City, request.CountryCode);

        try
        {
            // Try cache first
            var cached = await _cache.GetAsync<WeatherDto>(cacheKey, cancellationToken);
            if (cached != null)
            {
                _logger.LogDebug("Cache hit for weather in {City}", request.City);
                return ApiResponse<WeatherDto>.Ok(cached, "From cache");
            }

            // Check if we have recent data in the database
            if (await _repository.HasRecentDataAsync(request.City, _dataFreshness, cancellationToken))
            {
                var dbRecord = await _repository.GetLatestAsync(request.City, cancellationToken);
                if (dbRecord != null)
                {
                    var dto = MapToDto(dbRecord);
                    await _cache.SetAsync(cacheKey, dto, _cacheExpiration, cancellationToken);
                    return ApiResponse<WeatherDto>.Ok(dto, "From database");
                }
            }

            // Fetch from external API
            return await FetchAndStoreWeatherAsync(request, cacheKey, cancellationToken);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error getting weather for {City}", request.City);
            return ApiResponse<WeatherDto>.Fail($"Failed to get weather: {ex.Message}");
        }
    }

    public async Task<ApiResponse<WeatherDto>> RefreshWeatherAsync(
        WeatherRequest request,
        CancellationToken cancellationToken = default)
    {
        ArgumentNullException.ThrowIfNull(request);

        var cacheKey = BuildCacheKey(request.City, request.CountryCode);

        try
        {
            // Invalidate cache
            await _cache.RemoveAsync(cacheKey, cancellationToken);

            // Fetch fresh data from API
            return await FetchAndStoreWeatherAsync(request, cacheKey, cancellationToken);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error refreshing weather for {City}", request.City);
            return ApiResponse<WeatherDto>.Fail($"Failed to refresh weather: {ex.Message}");
        }
    }

    public async Task<ApiResponse<PaginatedResponse<WeatherDto>>> GetWeatherHistoryAsync(
        WeatherHistoryQuery query,
        int page = 1,
        int pageSize = 20,
        CancellationToken cancellationToken = default)
    {
        ArgumentNullException.ThrowIfNull(query);

        try
        {
            var records = await _repository.GetHistoryAsync(query, cancellationToken);
            var recordList = records.ToList();

            var totalCount = recordList.Count;
            var paginatedRecords = recordList
                .Skip((page - 1) * pageSize)
                .Take(pageSize)
                .Select(MapToDto)
                .ToList();

            var response = new PaginatedResponse<WeatherDto>
            {
                Items = paginatedRecords,
                Page = page,
                PageSize = pageSize,
                TotalCount = totalCount
            };

            return ApiResponse<PaginatedResponse<WeatherDto>>.Ok(response);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error getting weather history for {City}", query.City);
            return ApiResponse<PaginatedResponse<WeatherDto>>.Fail($"Failed to get history: {ex.Message}");
        }
    }

    public async Task<ApiResponse<WeatherStatistics>> GetWeatherStatisticsAsync(
        string city,
        DateTime? startDate = null,
        DateTime? endDate = null,
        CancellationToken cancellationToken = default)
    {
        ArgumentException.ThrowIfNullOrWhiteSpace(city);

        try
        {
            var stats = await _repository.GetStatisticsAsync(city, startDate, endDate, cancellationToken);
            if (stats == null)
            {
                return ApiResponse<WeatherStatistics>.Fail($"No weather data found for {city}");
            }

            return ApiResponse<WeatherStatistics>.Ok(stats);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error getting weather statistics for {City}", city);
            return ApiResponse<WeatherStatistics>.Fail($"Failed to get statistics: {ex.Message}");
        }
    }

    public async Task<ApiResponse<int>> CleanupOldRecordsAsync(
        int daysToKeep = 30,
        CancellationToken cancellationToken = default)
    {
        try
        {
            var cutoffDate = DateTime.UtcNow.AddDays(-daysToKeep);
            var deletedCount = await _repository.DeleteOlderThanAsync(cutoffDate, cancellationToken);

            _logger.LogInformation("Cleaned up {Count} old weather records", deletedCount);
            return ApiResponse<int>.Ok(deletedCount, $"Deleted {deletedCount} records older than {daysToKeep} days");
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error cleaning up old weather records");
            return ApiResponse<int>.Fail($"Failed to cleanup: {ex.Message}");
        }
    }

    private async Task<ApiResponse<WeatherDto>> FetchAndStoreWeatherAsync(
        WeatherRequest request,
        string cacheKey,
        CancellationToken cancellationToken)
    {
        var weatherData = await _apiClient.GetCurrentWeatherAsync(
            request.City,
            request.CountryCode,
            cancellationToken);

        if (weatherData == null)
        {
            return ApiResponse<WeatherDto>.Fail($"Weather data not found for {request.City}");
        }

        // Store in database
        var record = MapToRecord(weatherData);
        await _repository.AddAsync(record, cancellationToken);

        // Update cache
        await _cache.SetAsync(cacheKey, weatherData, _cacheExpiration, cancellationToken);

        _logger.LogInformation("Fetched and stored weather for {City}", request.City);
        return ApiResponse<WeatherDto>.Ok(weatherData, "From API");
    }

    private static string BuildCacheKey(string city, string? countryCode) =>
        string.IsNullOrEmpty(countryCode)
            ? $"weather:{city.ToLowerInvariant()}"
            : $"weather:{city.ToLowerInvariant()}:{countryCode.ToLowerInvariant()}";

    private static WeatherDto MapToDto(WeatherRecord record) => new(
        City: record.City,
        Country: record.CountryCode,
        Temperature: record.Temperature,
        FeelsLike: record.FeelsLike,
        Humidity: record.Humidity,
        WindSpeed: record.WindSpeed,
        Description: record.Description ?? "Unknown",
        RecordedAt: record.RecordedAt
    );

    private static WeatherRecord MapToRecord(WeatherDto dto) => new()
    {
        City = dto.City,
        CountryCode = dto.Country,
        Temperature = dto.Temperature,
        FeelsLike = dto.FeelsLike,
        Humidity = dto.Humidity,
        WindSpeed = dto.WindSpeed,
        Description = dto.Description,
        RecordedAt = dto.RecordedAt
    };
}
