using CSharpService.Core.Interfaces;
using CSharpService.Infrastructure.Data;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;

namespace CSharpService.Api.Controllers;

/// <summary>
/// Controller for health and diagnostic endpoints
/// </summary>
[ApiController]
[Route("api/[controller]")]
[Produces("application/json")]
public class HealthController : ControllerBase
{
    private readonly AppDbContext _dbContext;
    private readonly IWeatherApiClient _apiClient;
    private readonly ILogger<HealthController> _logger;

    public HealthController(
        AppDbContext dbContext,
        IWeatherApiClient apiClient,
        ILogger<HealthController> logger)
    {
        _dbContext = dbContext;
        _apiClient = apiClient;
        _logger = logger;
    }

    /// <summary>
    /// Gets detailed health status of all service dependencies
    /// </summary>
    [HttpGet("detailed")]
    [ProducesResponseType(typeof(DetailedHealthResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(DetailedHealthResponse), StatusCodes.Status503ServiceUnavailable)]
    public async Task<ActionResult<DetailedHealthResponse>> GetDetailedHealth(
        CancellationToken cancellationToken = default)
    {
        var response = new DetailedHealthResponse
        {
            Timestamp = DateTime.UtcNow,
            Version = GetType().Assembly.GetName().Version?.ToString() ?? "1.0.0"
        };

        // Check database connectivity
        response.Database = await CheckDatabaseHealthAsync(cancellationToken);

        // Check external API connectivity
        response.ExternalApi = await CheckExternalApiHealthAsync(cancellationToken);

        // Overall status
        response.Status = response.Database.IsHealthy && response.ExternalApi.IsHealthy
            ? HealthStatus.Healthy
            : HealthStatus.Unhealthy;

        var statusCode = response.Status == HealthStatus.Healthy
            ? StatusCodes.Status200OK
            : StatusCodes.Status503ServiceUnavailable;

        return StatusCode(statusCode, response);
    }

    /// <summary>
    /// Simple liveness probe
    /// </summary>
    [HttpGet("live")]
    [ProducesResponseType(StatusCodes.Status200OK)]
    public IActionResult GetLiveness()
    {
        return Ok(new { status = "alive", timestamp = DateTime.UtcNow });
    }

    /// <summary>
    /// Readiness probe with database check
    /// </summary>
    [HttpGet("ready")]
    [ProducesResponseType(StatusCodes.Status200OK)]
    [ProducesResponseType(StatusCodes.Status503ServiceUnavailable)]
    public async Task<IActionResult> GetReadiness(CancellationToken cancellationToken = default)
    {
        var dbHealth = await CheckDatabaseHealthAsync(cancellationToken);

        if (!dbHealth.IsHealthy)
        {
            return StatusCode(
                StatusCodes.Status503ServiceUnavailable,
                new { status = "not ready", reason = dbHealth.Message });
        }

        return Ok(new { status = "ready", timestamp = DateTime.UtcNow });
    }

    private async Task<ComponentHealth> CheckDatabaseHealthAsync(CancellationToken cancellationToken)
    {
        var health = new ComponentHealth { Name = "MySQL Database" };
        var stopwatch = System.Diagnostics.Stopwatch.StartNew();

        try
        {
            // Try to execute a simple query
            await _dbContext.Database.ExecuteSqlRawAsync("SELECT 1", cancellationToken);
            stopwatch.Stop();

            health.IsHealthy = true;
            health.Message = "Connected successfully";
            health.ResponseTimeMs = stopwatch.ElapsedMilliseconds;
        }
        catch (Exception ex)
        {
            stopwatch.Stop();
            health.IsHealthy = false;
            health.Message = $"Connection failed: {ex.Message}";
            health.ResponseTimeMs = stopwatch.ElapsedMilliseconds;
            _logger.LogError(ex, "Database health check failed");
        }

        return health;
    }

    private async Task<ComponentHealth> CheckExternalApiHealthAsync(CancellationToken cancellationToken)
    {
        var health = new ComponentHealth { Name = "OpenWeatherMap API" };
        var stopwatch = System.Diagnostics.Stopwatch.StartNew();

        try
        {
            var isValid = await _apiClient.ValidateApiKeyAsync(cancellationToken);
            stopwatch.Stop();

            health.IsHealthy = isValid;
            health.Message = isValid ? "API key valid" : "API key invalid or not configured";
            health.ResponseTimeMs = stopwatch.ElapsedMilliseconds;
        }
        catch (Exception ex)
        {
            stopwatch.Stop();
            health.IsHealthy = false;
            health.Message = $"API check failed: {ex.Message}";
            health.ResponseTimeMs = stopwatch.ElapsedMilliseconds;
            _logger.LogError(ex, "External API health check failed");
        }

        return health;
    }
}

#region Response Models

public class DetailedHealthResponse
{
    public HealthStatus Status { get; set; }
    public DateTime Timestamp { get; set; }
    public string Version { get; set; } = string.Empty;
    public ComponentHealth Database { get; set; } = new();
    public ComponentHealth ExternalApi { get; set; } = new();
}

public class ComponentHealth
{
    public string Name { get; set; } = string.Empty;
    public bool IsHealthy { get; set; }
    public string Message { get; set; } = string.Empty;
    public long ResponseTimeMs { get; set; }
}

public enum HealthStatus
{
    Healthy,
    Degraded,
    Unhealthy
}

#endregion
