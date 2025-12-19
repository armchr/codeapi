using System.Net.Http.Json;
using System.Text.Json;
using System.Text.Json.Serialization;
using CSharpService.Core.Interfaces;
using CSharpService.Core.Models;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;

namespace CSharpService.Infrastructure.External;

/// <summary>
/// Configuration for OpenWeatherMap API
/// </summary>
public class OpenWeatherApiConfig
{
    public const string SectionName = "OpenWeatherApi";

    public string ApiKey { get; set; } = string.Empty;
    public string BaseUrl { get; set; } = "https://api.openweathermap.org/data/2.5";
    public string Units { get; set; } = "metric";
    public int TimeoutSeconds { get; set; } = 30;
    public int RetryCount { get; set; } = 3;
}

/// <summary>
/// Client for OpenWeatherMap API
/// </summary>
public class OpenWeatherApiClient : IWeatherApiClient
{
    private readonly HttpClient _httpClient;
    private readonly OpenWeatherApiConfig _config;
    private readonly ILogger<OpenWeatherApiClient> _logger;
    private readonly JsonSerializerOptions _jsonOptions;

    public OpenWeatherApiClient(
        HttpClient httpClient,
        IOptions<OpenWeatherApiConfig> config,
        ILogger<OpenWeatherApiClient> logger)
    {
        _httpClient = httpClient ?? throw new ArgumentNullException(nameof(httpClient));
        _config = config?.Value ?? throw new ArgumentNullException(nameof(config));
        _logger = logger ?? throw new ArgumentNullException(nameof(logger));

        _jsonOptions = new JsonSerializerOptions
        {
            PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower,
            PropertyNameCaseInsensitive = true
        };

        ConfigureHttpClient();
    }

    private void ConfigureHttpClient()
    {
        _httpClient.BaseAddress = new Uri(_config.BaseUrl);
        _httpClient.Timeout = TimeSpan.FromSeconds(_config.TimeoutSeconds);
        _httpClient.DefaultRequestHeaders.Add("Accept", "application/json");
    }

    public async Task<WeatherDto?> GetCurrentWeatherAsync(
        string city,
        string? countryCode = null,
        CancellationToken cancellationToken = default)
    {
        var query = BuildLocationQuery(city, countryCode);
        var url = $"/weather?q={query}&units={_config.Units}&appid={_config.ApiKey}";

        _logger.LogDebug("Fetching weather from OpenWeatherMap for {City}", city);

        try
        {
            var response = await ExecuteWithRetryAsync(
                () => _httpClient.GetAsync(url, cancellationToken),
                cancellationToken);

            if (!response.IsSuccessStatusCode)
            {
                var errorContent = await response.Content.ReadAsStringAsync(cancellationToken);
                _logger.LogWarning(
                    "OpenWeatherMap API returned {StatusCode} for {City}: {Error}",
                    response.StatusCode, city, errorContent);
                return null;
            }

            var apiResponse = await response.Content.ReadFromJsonAsync<OpenWeatherResponse>(
                _jsonOptions,
                cancellationToken);

            if (apiResponse == null)
            {
                _logger.LogWarning("Received null response from OpenWeatherMap for {City}", city);
                return null;
            }

            return MapToWeatherDto(apiResponse);
        }
        catch (HttpRequestException ex)
        {
            _logger.LogError(ex, "HTTP error fetching weather for {City}", city);
            throw;
        }
        catch (TaskCanceledException ex) when (ex.InnerException is TimeoutException)
        {
            _logger.LogError(ex, "Timeout fetching weather for {City}", city);
            throw;
        }
    }

    public async Task<IEnumerable<WeatherDto>> GetForecastAsync(
        string city,
        int days = 5,
        CancellationToken cancellationToken = default)
    {
        var url = $"/forecast?q={Uri.EscapeDataString(city)}&cnt={days * 8}&units={_config.Units}&appid={_config.ApiKey}";

        _logger.LogDebug("Fetching {Days}-day forecast from OpenWeatherMap for {City}", days, city);

        try
        {
            var response = await ExecuteWithRetryAsync(
                () => _httpClient.GetAsync(url, cancellationToken),
                cancellationToken);

            if (!response.IsSuccessStatusCode)
            {
                _logger.LogWarning("OpenWeatherMap forecast API returned {StatusCode}", response.StatusCode);
                return Enumerable.Empty<WeatherDto>();
            }

            var forecastResponse = await response.Content.ReadFromJsonAsync<OpenWeatherForecastResponse>(
                _jsonOptions,
                cancellationToken);

            if (forecastResponse?.List == null)
            {
                return Enumerable.Empty<WeatherDto>();
            }

            return forecastResponse.List
                .Select(item => MapForecastItemToDto(item, forecastResponse.City))
                .ToList();
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error fetching forecast for {City}", city);
            return Enumerable.Empty<WeatherDto>();
        }
    }

    public async Task<bool> ValidateApiKeyAsync(CancellationToken cancellationToken = default)
    {
        if (string.IsNullOrWhiteSpace(_config.ApiKey))
        {
            return false;
        }

        try
        {
            // Make a simple request to validate the API key
            var url = $"/weather?q=London&appid={_config.ApiKey}";
            var response = await _httpClient.GetAsync(url, cancellationToken);
            return response.IsSuccessStatusCode;
        }
        catch
        {
            return false;
        }
    }

    private async Task<HttpResponseMessage> ExecuteWithRetryAsync(
        Func<Task<HttpResponseMessage>> action,
        CancellationToken cancellationToken)
    {
        var retryCount = 0;
        var maxRetries = _config.RetryCount;

        while (true)
        {
            try
            {
                var response = await action();

                // Don't retry on client errors (4xx)
                if ((int)response.StatusCode >= 400 && (int)response.StatusCode < 500)
                {
                    return response;
                }

                // Retry on server errors (5xx) if we have retries left
                if (!response.IsSuccessStatusCode && retryCount < maxRetries)
                {
                    retryCount++;
                    var delay = TimeSpan.FromSeconds(Math.Pow(2, retryCount));
                    _logger.LogWarning(
                        "Request failed with {StatusCode}, retrying in {Delay}s (attempt {Attempt}/{MaxRetries})",
                        response.StatusCode, delay.TotalSeconds, retryCount, maxRetries);
                    await Task.Delay(delay, cancellationToken);
                    continue;
                }

                return response;
            }
            catch (HttpRequestException) when (retryCount < maxRetries)
            {
                retryCount++;
                var delay = TimeSpan.FromSeconds(Math.Pow(2, retryCount));
                _logger.LogWarning(
                    "Request failed, retrying in {Delay}s (attempt {Attempt}/{MaxRetries})",
                    delay.TotalSeconds, retryCount, maxRetries);
                await Task.Delay(delay, cancellationToken);
            }
        }
    }

    private static string BuildLocationQuery(string city, string? countryCode) =>
        string.IsNullOrEmpty(countryCode)
            ? Uri.EscapeDataString(city)
            : Uri.EscapeDataString($"{city},{countryCode}");

    private static WeatherDto MapToWeatherDto(OpenWeatherResponse response) => new(
        City: response.Name,
        Country: response.Sys?.Country ?? "Unknown",
        Temperature: response.Main.Temp,
        FeelsLike: response.Main.FeelsLike,
        Humidity: response.Main.Humidity,
        WindSpeed: response.Wind.Speed,
        Description: response.Weather.FirstOrDefault()?.Description ?? "Unknown",
        RecordedAt: DateTimeOffset.FromUnixTimeSeconds(response.Dt).UtcDateTime
    );

    private static WeatherDto MapForecastItemToDto(ForecastItem item, CityInfo city) => new(
        City: city.Name,
        Country: city.Country,
        Temperature: item.Main.Temp,
        FeelsLike: item.Main.FeelsLike,
        Humidity: item.Main.Humidity,
        WindSpeed: item.Wind.Speed,
        Description: item.Weather.FirstOrDefault()?.Description ?? "Unknown",
        RecordedAt: DateTimeOffset.FromUnixTimeSeconds(item.Dt).UtcDateTime
    );
}

#region API Response Models

internal class OpenWeatherResponse
{
    public int Dt { get; set; }
    public string Name { get; set; } = string.Empty;
    public MainInfo Main { get; set; } = new();
    public WindInfo Wind { get; set; } = new();
    public List<WeatherInfo> Weather { get; set; } = new();
    public SysInfo? Sys { get; set; }
}

internal class OpenWeatherForecastResponse
{
    public List<ForecastItem> List { get; set; } = new();
    public CityInfo City { get; set; } = new();
}

internal class ForecastItem
{
    public int Dt { get; set; }
    public MainInfo Main { get; set; } = new();
    public WindInfo Wind { get; set; } = new();
    public List<WeatherInfo> Weather { get; set; } = new();
}

internal class MainInfo
{
    public double Temp { get; set; }

    [JsonPropertyName("feels_like")]
    public double FeelsLike { get; set; }

    public int Humidity { get; set; }
    public int Pressure { get; set; }
}

internal class WindInfo
{
    public double Speed { get; set; }
    public int Deg { get; set; }
}

internal class WeatherInfo
{
    public int Id { get; set; }
    public string Main { get; set; } = string.Empty;
    public string Description { get; set; } = string.Empty;
    public string Icon { get; set; } = string.Empty;
}

internal class SysInfo
{
    public string Country { get; set; } = string.Empty;
    public int Sunrise { get; set; }
    public int Sunset { get; set; }
}

internal class CityInfo
{
    public string Name { get; set; } = string.Empty;
    public string Country { get; set; } = string.Empty;
}

#endregion
