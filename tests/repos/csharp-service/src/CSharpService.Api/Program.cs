using CSharpService.Core.Interfaces;
using CSharpService.Core.Services;
using CSharpService.Infrastructure.Data;
using CSharpService.Infrastructure.External;
using CSharpService.Infrastructure.Repositories;
using Microsoft.EntityFrameworkCore;

var builder = WebApplication.CreateBuilder(args);

// Configure services
ConfigureServices(builder.Services, builder.Configuration);

var app = builder.Build();

// Configure middleware pipeline
ConfigureMiddleware(app);

// Initialize database
await InitializeDatabaseAsync(app);

app.Run();

void ConfigureServices(IServiceCollection services, IConfiguration configuration)
{
    // Add controllers
    services.AddControllers();

    // Add Swagger/OpenAPI
    services.AddEndpointsApiExplorer();
    services.AddSwaggerGen(options =>
    {
        options.SwaggerDoc("v1", new()
        {
            Title = "Weather Service API",
            Version = "v1",
            Description = "A sample weather service demonstrating C# patterns with MySQL and external API integration"
        });
    });

    // Configure MySQL with Entity Framework Core
    var connectionString = configuration.GetConnectionString("DefaultConnection")
        ?? "Server=localhost;Database=weather_db;User=root;Password=password;";

    services.AddDbContext<AppDbContext>(options =>
    {
        options.UseMySql(
            connectionString,
            ServerVersion.AutoDetect(connectionString),
            mysqlOptions =>
            {
                mysqlOptions.EnableRetryOnFailure(
                    maxRetryCount: 3,
                    maxRetryDelay: TimeSpan.FromSeconds(30),
                    errorNumbersToAdd: null);
                mysqlOptions.CommandTimeout(30);
            });
    });

    // Configure OpenWeatherMap API client
    services.Configure<OpenWeatherApiConfig>(
        configuration.GetSection(OpenWeatherApiConfig.SectionName));

    services.AddHttpClient<IWeatherApiClient, OpenWeatherApiClient>(client =>
    {
        client.DefaultRequestHeaders.Add("User-Agent", "CSharpService/1.0");
    })
    .ConfigurePrimaryHttpMessageHandler(() => new HttpClientHandler
    {
        AutomaticDecompression = System.Net.DecompressionMethods.GZip | System.Net.DecompressionMethods.Deflate
    });

    // Register application services
    services.AddScoped<IWeatherRepository, WeatherRepository>();
    services.AddScoped<IWeatherService, WeatherService>();
    services.AddSingleton<ICacheService, MemoryCacheService>();

    // Add health checks
    services.AddHealthChecks()
        .AddDbContextCheck<AppDbContext>("database");

    // Add logging
    services.AddLogging(logging =>
    {
        logging.AddConsole();
        logging.AddDebug();
    });
}

void ConfigureMiddleware(WebApplication app)
{
    if (app.Environment.IsDevelopment())
    {
        app.UseSwagger();
        app.UseSwaggerUI(options =>
        {
            options.SwaggerEndpoint("/swagger/v1/swagger.json", "Weather Service API v1");
            options.RoutePrefix = string.Empty;
        });
    }

    app.UseHttpsRedirection();
    app.UseAuthorization();
    app.MapControllers();
    app.MapHealthChecks("/health");
}

async Task InitializeDatabaseAsync(WebApplication app)
{
    using var scope = app.Services.CreateScope();
    var context = scope.ServiceProvider.GetRequiredService<AppDbContext>();
    var logger = scope.ServiceProvider.GetRequiredService<ILogger<Program>>();

    try
    {
        logger.LogInformation("Applying database migrations...");
        await context.Database.MigrateAsync();
        logger.LogInformation("Database migrations applied successfully");
    }
    catch (Exception ex)
    {
        logger.LogError(ex, "Error applying database migrations");
        throw;
    }
}
