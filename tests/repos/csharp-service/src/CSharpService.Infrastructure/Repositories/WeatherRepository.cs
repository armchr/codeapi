using CSharpService.Core.Interfaces;
using CSharpService.Core.Models;
using CSharpService.Infrastructure.Data;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging;

namespace CSharpService.Infrastructure.Repositories;

/// <summary>
/// Entity Framework implementation of weather repository
/// </summary>
public class WeatherRepository : IWeatherRepository
{
    private readonly AppDbContext _context;
    private readonly ILogger<WeatherRepository> _logger;

    public WeatherRepository(AppDbContext context, ILogger<WeatherRepository> logger)
    {
        _context = context ?? throw new ArgumentNullException(nameof(context));
        _logger = logger ?? throw new ArgumentNullException(nameof(logger));
    }

    public async Task<WeatherRecord?> GetLatestAsync(string city, CancellationToken cancellationToken = default)
    {
        return await _context.WeatherRecords
            .Where(w => w.City.ToLower() == city.ToLower())
            .OrderByDescending(w => w.RecordedAt)
            .FirstOrDefaultAsync(cancellationToken);
    }

    public async Task<IEnumerable<WeatherRecord>> GetHistoryAsync(
        WeatherHistoryQuery query,
        CancellationToken cancellationToken = default)
    {
        var queryable = _context.WeatherRecords
            .Where(w => w.City.ToLower() == query.City.ToLower());

        if (query.StartDate.HasValue)
        {
            queryable = queryable.Where(w => w.RecordedAt >= query.StartDate.Value);
        }

        if (query.EndDate.HasValue)
        {
            queryable = queryable.Where(w => w.RecordedAt <= query.EndDate.Value);
        }

        return await queryable
            .OrderByDescending(w => w.RecordedAt)
            .Take(query.Limit)
            .ToListAsync(cancellationToken);
    }

    public async Task<WeatherStatistics?> GetStatisticsAsync(
        string city,
        DateTime? startDate = null,
        DateTime? endDate = null,
        CancellationToken cancellationToken = default)
    {
        var query = _context.WeatherRecords
            .Where(w => w.City.ToLower() == city.ToLower());

        if (startDate.HasValue)
        {
            query = query.Where(w => w.RecordedAt >= startDate.Value);
        }

        if (endDate.HasValue)
        {
            query = query.Where(w => w.RecordedAt <= endDate.Value);
        }

        var records = await query.ToListAsync(cancellationToken);

        if (records.Count == 0)
        {
            return null;
        }

        return new WeatherStatistics(
            City: city,
            AverageTemperature: records.Average(r => r.Temperature),
            MinTemperature: records.Min(r => r.Temperature),
            MaxTemperature: records.Max(r => r.Temperature),
            AverageHumidity: records.Average(r => r.Humidity),
            RecordCount: records.Count,
            OldestRecord: records.Min(r => r.RecordedAt),
            NewestRecord: records.Max(r => r.RecordedAt)
        );
    }

    public async Task<WeatherRecord> AddAsync(WeatherRecord record, CancellationToken cancellationToken = default)
    {
        record.CreatedAt = DateTime.UtcNow;
        _context.WeatherRecords.Add(record);
        await _context.SaveChangesAsync(cancellationToken);

        _logger.LogDebug("Added weather record for {City} with Id {Id}", record.City, record.Id);
        return record;
    }

    public async Task<int> AddBulkAsync(IEnumerable<WeatherRecord> records, CancellationToken cancellationToken = default)
    {
        var recordList = records.ToList();
        var now = DateTime.UtcNow;

        foreach (var record in recordList)
        {
            record.CreatedAt = now;
        }

        await _context.WeatherRecords.AddRangeAsync(recordList, cancellationToken);
        var count = await _context.SaveChangesAsync(cancellationToken);

        _logger.LogInformation("Added {Count} weather records in bulk", count);
        return count;
    }

    public async Task<int> DeleteOlderThanAsync(DateTime cutoffDate, CancellationToken cancellationToken = default)
    {
        var count = await _context.WeatherRecords
            .Where(w => w.RecordedAt < cutoffDate)
            .ExecuteDeleteAsync(cancellationToken);

        _logger.LogInformation("Deleted {Count} weather records older than {CutoffDate}", count, cutoffDate);
        return count;
    }

    public async Task<bool> HasRecentDataAsync(string city, TimeSpan maxAge, CancellationToken cancellationToken = default)
    {
        var cutoff = DateTime.UtcNow - maxAge;
        return await _context.WeatherRecords
            .AnyAsync(w => w.City.ToLower() == city.ToLower() && w.RecordedAt >= cutoff, cancellationToken);
    }
}
