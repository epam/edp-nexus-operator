package webhook

import (
	"encoding/json"
	"fmt"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

func validateCreate(spec *nexusApi.NexusRepositorySpec) error {
	specm, err := specToMap(spec)
	if err != nil {
		return fmt.Errorf("unable to convert spec to map: %w", err)
	}

	if len(specm) > 1 {
		return fmt.Errorf("repository must have only one format - go, maven, npm, etc")
	}

	if len(specm) == 0 {
		return fmt.Errorf("repository format is not specified")
	}

	for _, v := range specm {
		if len(v) > 1 {
			return fmt.Errorf("repository must have only one type - hosted, proxy or group")
		}

		if len(v) == 0 {
			return fmt.Errorf("repository type is not specified")
		}
	}

	return nil
}

func validateUpdate(oldSpec, newSpec *nexusApi.NexusRepositorySpec) error {
	if err := validateCreate(newSpec); err != nil {
		return err
	}

	oldspecm, err := specToMap(oldSpec)
	if err != nil {
		return fmt.Errorf("unable to convert spec to map: %w", err)
	}

	newspecm, err := specToMap(newSpec)
	if err != nil {
		return fmt.Errorf("unable to convert spec to map: %w", err)
	}

	for oldFormat, oldFormatVal := range oldspecm {
		val, ok := newspecm[oldFormat]
		if !ok {
			return fmt.Errorf("repository format %s cannot be changed to another", oldFormat)
		}

		for oldType := range oldFormatVal {
			_, ok = val[oldType]
			if !ok {
				return fmt.Errorf("repository type %s cannot be changed to another", oldType)
			}
		}
	}

	return nil
}

func specToMap(spec *nexusApi.NexusRepositorySpec) (map[string]map[string]interface{}, error) {
	specj, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal spec: %w", err)
	}

	specm := map[string]map[string]interface{}{}
	if err = json.Unmarshal(specj, &specm); err != nil {
		return nil, fmt.Errorf("unable to unmarshal spec: %w", err)
	}

	delete(specm, "nexusRef")

	return specm, nil
}
