package vectorstore

import (
	"context"
	"fmt"

	qdrant "github.com/qdrant/go-client/qdrant"
)

// DescriptionKey is the collection metadata key holding a human-readable
// summary of the collection's contents. Query-side tools read it to decide
// which collection to search.
const DescriptionKey = "description"

// SetDescription writes description into the collection's metadata. It
// returns exists=false when the collection hasn't been created yet, and
// changed=false when the stored description already matches.
func SetDescription(ctx context.Context, client *qdrant.Client, name, description string) (exists, changed bool, err error) {
	exists, err = client.CollectionExists(ctx, name)
	if err != nil {
		return false, false, fmt.Errorf("check collection: %w", err)
	}
	if !exists {
		return false, false, nil
	}

	info, err := client.GetCollectionInfo(ctx, name)
	if err != nil {
		return true, false, fmt.Errorf("get collection info: %w", err)
	}
	if Description(info) == description {
		return true, false, nil
	}

	if err := client.UpdateCollection(ctx, &qdrant.UpdateCollection{
		CollectionName: name,
		Metadata:       map[string]*qdrant.Value{DescriptionKey: qdrant.NewValueString(description)},
	}); err != nil {
		return true, false, fmt.Errorf("update collection metadata: %w", err)
	}
	return true, true, nil
}

// Description returns the description stored in a collection's metadata, or
// "" when none is set.
func Description(info *qdrant.CollectionInfo) string {
	return info.GetConfig().GetMetadata()[DescriptionKey].GetStringValue()
}
