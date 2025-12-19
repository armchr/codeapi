using CSharpService.Core.Models;

namespace CSharpService.Core.Interfaces;

/// <summary>
/// Repository interface for weather data persistence
/// </summary>
public interface IWeatherRepository
{
    /// <summary>
    /// Gets the latest weather record for a city
    /// </summary>
    Task<WeatherRecord?> GetLatestAsync(string city, CancellationToken cancellationToken = default);

    /// <summary>
    /// Gets weather history for a city
    /// </summary>
    Task<IEnumerable<WeatherRecord>> GetHistoryAsync(
        WeatherHistoryQuery query,
        CancellationToken cancellationToken = default);

    /// <summary>
    /// Gets weather statistics for a city
    /// </summary>
    Task<WeatherStatistics?> GetStatisticsAsync(
        string city,
        DateTime? startDate = null,
        DateTime? endDate = null,
        CancellationToken cancellationToken = default);

    /// <summary>
    /// Adds a new weather record
    /// </summary>
    Task<WeatherRecord> AddAsync(WeatherRecord record, CancellationToken cancellationToken = default);

    /// <summary>
    /// Adds multiple weather records in bulk
    /// </summary>
    Task<int> AddBulkAsync(IEnumerable<WeatherRecord> records, CancellationToken cancellationToken = default);

    /// <summary>
    /// Deletes old weather records
    /// </summary>
    Task<int> DeleteOlderThanAsync(DateTime cutoffDate, CancellationToken cancellationToken = default);

    /// <summary>
    /// Checks if recent data exists for a city
    /// </summary>
    Task<bool> HasRecentDataAsync(string city, TimeSpan maxAge, CancellationToken cancellationToken = default);
}

/// <summary>
/// Interface for external weather API client
/// </summary>
public interface IWeatherApiClient
{
    /// <summary>
    /// Fetches current weather from external API
    /// </summary>
    Task<WeatherDto?> GetCurrentWeatherAsync(
        string city,
        string? countryCode = null,
        CancellationToken cancellationToken = default);

    /// <summary>
    /// Fetches weather forecast from external API
    /// </summary>
    Task<IEnumerable<WeatherDto>> GetForecastAsync(
        string city,
        int days = 5,
        CancellationToken cancellationToken = default);

    /// <summary>
    /// Validates if an API key is configured and valid
    /// </summary>
    Task<bool> ValidateApiKeyAsync(CancellationToken cancellationToken = default);
}
