# Design: AUR Package Update Notifications

## Overview
The goal is to add support for AUR package update notifications. Since the AUR RSS feed does not include the package version in its items, a version resolver abstraction is introduced to fetch versions from the AUR RPC API.

## Architecture

### Version Resolver Abstraction
A `VersionResolver` interface is introduced to decouple the extraction of package name and version from the feed item source.

```go
type VersionResolver interface {
    // Resolve extracts the package name and version from a feed item.
    Resolve(ctx context.Context, item *gofeed.Item) (pkg, version string, err error)
}
```

### Implementations

#### 1. `StandardResolver`
- **Purpose**: Handles official Arch Linux feeds.
- **Logic**: Uses the existing `titleRegex` (`^(\S+)\s+(\S+)`) to extract the package name and version from `item.Title`.

#### 2. `AURResolver`
- **Purpose**: Handles the AUR feed (`https://aur.archlinux.org/rss/modified`).
- **Logic**:
    1. Extracts the package name from `item.Title` (the AUR feed title typically starts with the package name).
    2. Calls the AUR RPC API: `https://aur.archlinux.org/rpc/v5/info/<pkg>`.
    3. Parses the JSON response.
    4. Extracts the `Version` field from the first element of the `results` array.

**AUR RPC JSON Structure:**
```json
{
  "results": [
    {
      "Name": "zoom",
      "Version": "7.1.0-1",
      ...
    }
  ],
  ...
}
```

## Components & Data Flow

### `reader.go`
`FetchFeeds` will be modified to return a `map[string][]*gofeed.Item` instead of a slice. The key is the feed URL, allowing the system to map each item to its corresponding resolver.

### `main.go` (`checkUpdates`)
The update loop is modified as follows:
1. Call `FetchFeeds` $\rightarrow$ get `map[url][]*gofeed.Item`.
2. For each `url` and its `items`:
    - Select the `VersionResolver` based on the `url`.
    - For each `item`:
        - Call `resolver.Resolve(ctx, item)`.
        - If `pkg` is installed and `version` differs from the installed version:
            - Check `state.ShouldNotify`.
            - Send notification and mark as notified in state.

## Error Handling & Testing

### Error Handling
- **API Failures**: Failures in the AUR RPC call are logged as warnings, and the specific package is skipped.
- **Parsing Failures**: If the package name cannot be extracted or JSON is malformed, the item is skipped.
- **Timeouts**: All HTTP calls use the shared `httpClient` with a 30s timeout.

### Testing
- **Unit Tests**:
    - `StandardResolver` tested with official feed samples.
    - `AURResolver` tested using a mock HTTP server providing the provided JSON sample.
- **Integration**: Verify the full flow from feed fetch to notification.
