using CSharpService.Core.Models;
using CSharpService.Core.Services;
using Microsoft.AspNetCore.Mvc;

namespace CSharpService.Api.Controllers;

/// <summary>
/// Controller for weather-related API endpoints
/// </summary>
[ApiController]
[Route("api/[controller]")]
[Produces("application/json")]
public class WeatherController : ControllerBase
{
    private readonly IWeatherService _weatherService;
    private readonly ILogger<WeatherController> _logger;

    public WeatherController(IWeatherService weatherService, ILogger<WeatherController> logger)
    {
        _weatherService = weatherService ?? throw new ArgumentNullException(nameof(weatherService));
        _logger = logger ?? throw new ArgumentNullException(nameof(logger));
    }

    /// <summary>
    /// Gets current weather for a city
    /// </summary>
    /// <param name="city">City name</param>
    /// <param name="countryCode">Optional ISO 3166 country code</param>
    /// <param name="cancellationToken">Cancellation token</param>
    /// <returns>Current weather data</returns>
    [HttpGet("{city}")]
    [ProducesResponseType(typeof(ApiResponse<WeatherDto>), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ApiResponse<WeatherDto>), StatusCodes.Status404NotFound)]
    [ProducesResponseType(typeof(ApiResponse<WeatherDto>), StatusCodes.Status500InternalServerError)]
    public async Task<ActionResult<ApiResponse<WeatherDto>>> GetCurrentWeather(
        string city,
        [FromQuery] string? countryCode = null,
        CancellationToken cancellationToken = default)
    {
        _logger.LogInformation("Getting current weather for {City}, {CountryCode}", city, countryCode);

        var request = new WeatherRequest(city, countryCode);
        var result = await _weatherService.GetCurrentWeatherAsync(request, cancellationToken);

        if (!result.Success)
        {
            return NotFound(result);
        }

        return Ok(result);
    }

    /// <summary>
    /// Forces a refresh of weather data from the external API
    /// </summary>
    [HttpPost("{city}/refresh")]
    [ProducesResponseType(typeof(ApiResponse<WeatherDto>), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ApiResponse<WeatherDto>), StatusCodes.Status404NotFound)]
    public async Task<ActionResult<ApiResponse<WeatherDto>>> RefreshWeather(
        string city,
        [FromQuery] string? countryCode = null,
        CancellationToken cancellationToken = default)
    {
        _logger.LogInformation("Refreshing weather for {City}", city);

        var request = new WeatherRequest(city, countryCode);
        var result = await _weatherService.RefreshWeatherAsync(request, cancellationToken);

        if (!result.Success)
        {
            return NotFound(result);
        }

        return Ok(result);
    }

    /// <summary>
    /// Gets historical weather data for a city
    /// </summary>
    [HttpGet("{city}/history")]
    [ProducesResponseType(typeof(ApiResponse<PaginatedResponse<WeatherDto>>), StatusCodes.Status200OK)]
    public async Task<ActionResult<ApiResponse<PaginatedResponse<WeatherDto>>>> GetWeatherHistory(
        string city,
        [FromQuery] DateTime? startDate = null,
        [FromQuery] DateTime? endDate = null,
        [FromQuery] int page = 1,
        [FromQuery] int pageSize = 20,
        [FromQuery] int limit = 100,
        CancellationToken cancellationToken = default)
    {
        _logger.LogInformation(
            "Getting weather history for {City} from {StartDate} to {EndDate}",
            city, startDate, endDate);

        var query = new WeatherHistoryQuery(city, startDate, endDate, limit);
        var result = await _weatherService.GetWeatherHistoryAsync(query, page, pageSize, cancellationToken);

        return Ok(result);
    }

    /// <summary>
    /// Gets weather statistics for a city
    /// </summary>
    [HttpGet("{city}/statistics")]
    [ProducesResponseType(typeof(ApiResponse<WeatherStatistics>), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ApiResponse<WeatherStatistics>), StatusCodes.Status404NotFound)]
    public async Task<ActionResult<ApiResponse<WeatherStatistics>>> GetWeatherStatistics(
        string city,
        [FromQuery] DateTime? startDate = null,
        [FromQuery] DateTime? endDate = null,
        CancellationToken cancellationToken = default)
    {
        _logger.LogInformation("Getting weather statistics for {City}", city);

        var result = await _weatherService.GetWeatherStatisticsAsync(
            city, startDate, endDate, cancellationToken);

        if (!result.Success)
        {
            return NotFound(result);
        }

        return Ok(result);
    }

    /// <summary>
    /// Cleans up old weather records
    /// </summary>
    [HttpDelete("cleanup")]
    [ProducesResponseType(typeof(ApiResponse<int>), StatusCodes.Status200OK)]
    public async Task<ActionResult<ApiResponse<int>>> CleanupOldRecords(
        [FromQuery] int daysToKeep = 30,
        CancellationToken cancellationToken = default)
    {
        _logger.LogInformation("Cleaning up weather records older than {Days} days", daysToKeep);

        var result = await _weatherService.CleanupOldRecordsAsync(daysToKeep, cancellationToken);
        return Ok(result);
    }
}
