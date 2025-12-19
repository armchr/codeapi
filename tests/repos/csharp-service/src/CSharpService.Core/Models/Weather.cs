using System.ComponentModel.DataAnnotations;
using System.ComponentModel.DataAnnotations.Schema;

namespace CSharpService.Core.Models;

/// <summary>
/// Represents weather data for a specific location
/// </summary>
public class WeatherRecord
{
    [Key]
    [DatabaseGenerated(DatabaseGeneratedOption.Identity)]
    public int Id { get; set; }

    [Required]
    [MaxLength(100)]
    public string City { get; set; } = string.Empty;

    [Required]
    [MaxLength(10)]
    public string CountryCode { get; set; } = string.Empty;

    public double Temperature { get; set; }

    public double FeelsLike { get; set; }

    public int Humidity { get; set; }

    public double WindSpeed { get; set; }

    [MaxLength(50)]
    public string? Description { get; set; }

    [MaxLength(20)]
    public string? Icon { get; set; }

    public DateTime RecordedAt { get; set; } = DateTime.UtcNow;

    public DateTime CreatedAt { get; set; } = DateTime.UtcNow;
}

/// <summary>
/// DTO for weather API responses
/// </summary>
public record WeatherDto(
    string City,
    string Country,
    double Temperature,
    double FeelsLike,
    int Humidity,
    double WindSpeed,
    string Description,
    DateTime RecordedAt
);

/// <summary>
/// Request model for fetching weather
/// </summary>
public record WeatherRequest(string City, string? CountryCode = null);

/// <summary>
/// Historical weather query parameters
/// </summary>
public record WeatherHistoryQuery(
    string City,
    DateTime? StartDate = null,
    DateTime? EndDate = null,
    int Limit = 100
);

/// <summary>
/// Statistics for weather data
/// </summary>
public record WeatherStatistics(
    string City,
    double AverageTemperature,
    double MinTemperature,
    double MaxTemperature,
    double AverageHumidity,
    int RecordCount,
    DateTime? OldestRecord,
    DateTime? NewestRecord
);
