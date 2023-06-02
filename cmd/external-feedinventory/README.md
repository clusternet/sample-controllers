# Sample External FeedInventory Controller

When using external `FeedInventory` controllers,
feature gate `FeedInventory` should be set to `false` on
`clusternet-controller-manager` side.

In the `FeedInventory` controller, we may register and plumb multiple
handlers. For each handler, we only need to implement below interfaces.

```go
// PluginFactory is an interface that must be implemented for each plugin.
type PluginFactory interface {
	// Parser parses the raw data to get the replicas, resource requirements, replica jsonpath, etc.
	Parser(rawData []byte) (*int32, appsapi.ReplicaRequirements, string, error)
	// Name returns name of the plugin. It is used in logs, etc.
	Name() string
	// Kind returns the resource kind.
	Kind() string
}
```

Please refer to
[Scheduling Requirement Insights](https://clusternet.io/docs/user-guide/clusternet-feed-inventory/)
to learn more about `FeedInventory`.
