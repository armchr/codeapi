using System.Collections.Concurrent;
using CSharpService.Core.Interfaces;
using Microsoft.Extensions.Logging;

namespace CSharpService.Infrastructure.External;

/// <summary>
/// Simple in-memory cache implementation
/// </summary>
public class MemoryCacheService : ICacheService
{
    private readonly ConcurrentDictionary<string, CacheEntry> _cache = new();
    private readonly ILogger<MemoryCacheService> _logger;
    private readonly SemaphoreSlim _cleanupLock = new(1, 1);
    private DateTime _lastCleanup = DateTime.UtcNow;
    private readonly TimeSpan _cleanupInterval = TimeSpan.FromMinutes(5);

    public MemoryCacheService(ILogger<MemoryCacheService> logger)
    {
        _logger = logger ?? throw new ArgumentNullException(nameof(logger));
    }

    public Task<T?> GetAsync<T>(string key, CancellationToken cancellationToken = default) where T : class
    {
        CleanupExpiredEntriesIfNeeded();

        if (_cache.TryGetValue(key, out var entry) && !entry.IsExpired)
        {
            _logger.LogDebug("Cache hit for key: {Key}", key);
            return Task.FromResult(entry.Value as T);
        }

        _logger.LogDebug("Cache miss for key: {Key}", key);
        return Task.FromResult<T?>(null);
    }

    public Task SetAsync<T>(
        string key,
        T value,
        TimeSpan? expiration = null,
        CancellationToken cancellationToken = default) where T : class
    {
        var entry = new CacheEntry(value, expiration);
        _cache.AddOrUpdate(key, entry, (_, _) => entry);

        _logger.LogDebug(
            "Cached key: {Key} with expiration: {Expiration}",
            key,
            entry.ExpiresAt?.ToString("o") ?? "never");

        return Task.CompletedTask;
    }

    public Task RemoveAsync(string key, CancellationToken cancellationToken = default)
    {
        _cache.TryRemove(key, out _);
        _logger.LogDebug("Removed cache key: {Key}", key);
        return Task.CompletedTask;
    }

    public async Task<T> GetOrCreateAsync<T>(
        string key,
        Func<Task<T>> factory,
        TimeSpan? expiration = null,
        CancellationToken cancellationToken = default) where T : class
    {
        var cached = await GetAsync<T>(key, cancellationToken);
        if (cached != null)
        {
            return cached;
        }

        var value = await factory();
        await SetAsync(key, value, expiration, cancellationToken);
        return value;
    }

    public Task<bool> ExistsAsync(string key, CancellationToken cancellationToken = default)
    {
        CleanupExpiredEntriesIfNeeded();
        var exists = _cache.TryGetValue(key, out var entry) && !entry.IsExpired;
        return Task.FromResult(exists);
    }

    private void CleanupExpiredEntriesIfNeeded()
    {
        if (DateTime.UtcNow - _lastCleanup < _cleanupInterval)
        {
            return;
        }

        if (!_cleanupLock.Wait(0))
        {
            return; // Another thread is already cleaning up
        }

        try
        {
            var expiredKeys = _cache
                .Where(kvp => kvp.Value.IsExpired)
                .Select(kvp => kvp.Key)
                .ToList();

            foreach (var key in expiredKeys)
            {
                _cache.TryRemove(key, out _);
            }

            if (expiredKeys.Count > 0)
            {
                _logger.LogDebug("Cleaned up {Count} expired cache entries", expiredKeys.Count);
            }

            _lastCleanup = DateTime.UtcNow;
        }
        finally
        {
            _cleanupLock.Release();
        }
    }

    private class CacheEntry
    {
        public object Value { get; }
        public DateTime? ExpiresAt { get; }
        public bool IsExpired => ExpiresAt.HasValue && DateTime.UtcNow > ExpiresAt.Value;

        public CacheEntry(object value, TimeSpan? expiration)
        {
            Value = value;
            ExpiresAt = expiration.HasValue ? DateTime.UtcNow.Add(expiration.Value) : null;
        }
    }
}
