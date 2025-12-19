using CSharpService.Core.Models;
using Microsoft.EntityFrameworkCore;

namespace CSharpService.Infrastructure.Data;

/// <summary>
/// Entity Framework Core database context for MySQL
/// </summary>
public class AppDbContext : DbContext
{
    public AppDbContext(DbContextOptions<AppDbContext> options) : base(options)
    {
    }

    public DbSet<WeatherRecord> WeatherRecords => Set<WeatherRecord>();

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        base.OnModelCreating(modelBuilder);

        ConfigureWeatherRecord(modelBuilder);
    }

    private static void ConfigureWeatherRecord(ModelBuilder modelBuilder)
    {
        modelBuilder.Entity<WeatherRecord>(entity =>
        {
            entity.ToTable("weather_records");

            entity.HasKey(e => e.Id);

            entity.Property(e => e.Id)
                .HasColumnName("id")
                .ValueGeneratedOnAdd();

            entity.Property(e => e.City)
                .HasColumnName("city")
                .HasMaxLength(100)
                .IsRequired();

            entity.Property(e => e.CountryCode)
                .HasColumnName("country_code")
                .HasMaxLength(10)
                .IsRequired();

            entity.Property(e => e.Temperature)
                .HasColumnName("temperature")
                .HasPrecision(5, 2);

            entity.Property(e => e.FeelsLike)
                .HasColumnName("feels_like")
                .HasPrecision(5, 2);

            entity.Property(e => e.Humidity)
                .HasColumnName("humidity");

            entity.Property(e => e.WindSpeed)
                .HasColumnName("wind_speed")
                .HasPrecision(5, 2);

            entity.Property(e => e.Description)
                .HasColumnName("description")
                .HasMaxLength(50);

            entity.Property(e => e.Icon)
                .HasColumnName("icon")
                .HasMaxLength(20);

            entity.Property(e => e.RecordedAt)
                .HasColumnName("recorded_at")
                .IsRequired();

            entity.Property(e => e.CreatedAt)
                .HasColumnName("created_at")
                .HasDefaultValueSql("CURRENT_TIMESTAMP");

            // Indexes for common queries
            entity.HasIndex(e => e.City)
                .HasDatabaseName("idx_weather_city");

            entity.HasIndex(e => new { e.City, e.RecordedAt })
                .HasDatabaseName("idx_weather_city_recorded");

            entity.HasIndex(e => e.RecordedAt)
                .HasDatabaseName("idx_weather_recorded_at");
        });
    }
}

/// <summary>
/// Database context factory for design-time migrations
/// </summary>
public class AppDbContextFactory : IDbContextFactory<AppDbContext>
{
    private readonly DbContextOptions<AppDbContext> _options;

    public AppDbContextFactory(DbContextOptions<AppDbContext> options)
    {
        _options = options;
    }

    public AppDbContext CreateDbContext()
    {
        return new AppDbContext(_options);
    }
}
