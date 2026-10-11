# datafs-client-go

Go client for a self-hosted DataFS install. MIT.

```go
client := &datafs.Client{
    BaseURL: "https://analytics.example.com",
    APIKey:  os.Getenv("DATAFS_API_KEY"),
}
series, err := client.Timeseries(ctx, siteID, "2026-09-12", "2026-10-11", "day")
```

`ServerEvent` uses an ingest token. The stats methods use a stats API key. See `API.md`.

```
go test ./...
```
