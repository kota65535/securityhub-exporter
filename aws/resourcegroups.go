package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"golang.org/x/sync/errgroup"
)

func GetResourcesTags(ctx context.Context, resourceIds []string) (ResourceID2Tags, error) {
	cfg, _ := config.LoadDefaultConfig(ctx)
	client := resourcegroupstaggingapi.NewFromConfig(cfg)

	resourceIdChunks := chunk(resourceIds, 100)

	errs, ctx := errgroup.WithContext(ctx)

	ch := make(chan []types.ResourceTagMapping, len(resourceIdChunks))

	for _, chunk := range resourceIdChunks {
		chunk := chunk
		errs.Go(func() error {
			input := resourcegroupstaggingapi.GetResourcesInput{
				ResourceARNList: chunk,
			}
			ret, err := client.GetResources(ctx, &input)
			if err != nil {
				return err
			}
			ch <- ret.ResourceTagMappingList
			return nil
		})
	}
	err := errs.Wait()
	if err != nil {
		return nil, err
	}

	result := make(ResourceID2Tags, 0)
	for i := 0; i < len(resourceIdChunks); i++ {
		mappings := <-ch
		for _, m := range mappings {
			result[*m.ResourceARN] = m.Tags
		}
	}

	return result, nil
}

func chunk[T any](slice []T, size int) [][]T {
	var chunks [][]T
	for i := 0; i < len(slice); {
		// Clamp the last chunk to the slice bound as necessary.
		end := size
		if l := len(slice[i:]); l < size {
			end = l
		}

		// Set the capacity of each chunk so that appending to a chunk does not
		// modify the original slice.
		chunks = append(chunks, slice[i:i+end:i+end])
		i += end
	}

	return chunks
}
