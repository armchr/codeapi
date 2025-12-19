using CSharpService.Core.Models;

namespace CSharpService.Core.Services;

/// <summary>
/// Weather service interface - orchestrates weather data operations
/// </summary>
public interface IWeatherService
{
    /// <summary>
    /// Gets current weather for a city, fetching from API if needed
    /// </summary>
    Task<ApiResponse<WeatherDto>> GetCurrentWeatherAsync(
        WeatherRequest request,
        CancellationToken cancellationToken = default);

    /// <summary>
    /// Gets weather history for a city from the database
    /// </summary>
    Task<ApiResponse<PaginatedResponse<WeatherDto>>> GetWeatherHistoryAsync(
        WeatherHistoryQuery query,
        int page = 1,
        int pageSize = 20,
        CancellationToken cancellationToken = default);

    /// <summary>
    /// Gets weather statistics for a city
    /// </summary>
    Task<ApiResponse<WeatherStatistics>> GetWeatherStatisticsAsync(
        string city,
        DateTime? startDate = null,
        DateTime? endDate = null,
        CancellationToken cancellationToken = default);

    /// <summary>
    /// Forces a refresh of weather data from the external API
    /// </summary>
    Task<ApiResponse<WeatherDto>> RefreshWeatherAsync(
        WeatherRequest request,
        CancellationToken cancellationToken = default);

    /// <summary>
    /// Cleans up old weather records
    /// </summary>
    Task<ApiResponse<int>> CleanupOldRecordsAsync(
        int daysToKeep = 30,
        CancellationToken cancellationToken = default);
}
