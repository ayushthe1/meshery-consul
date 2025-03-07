package oam

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/layer5io/meshery-adapter-library/adapter"
	"github.com/layer5io/meshery-consul/internal/config"
	"github.com/layer5io/meshkit/models/meshmodel/core/types"
)

var (
	basePath, _         = os.Getwd()
	WorkloadPath        = filepath.Join(basePath, "templates", "oam", "workloads")
	MeshmodelComponents = filepath.Join(basePath, "templates", "meshmodel", "components")
	traitPath           = filepath.Join(basePath, "templates", "oam", "traits")
)

type schemaDefinitionPathSet struct {
	oamDefinitionPath string
	jsonSchemaPath    string
	name              string
}

var AvailableVersions = map[string]bool{}
var pathSets []schemaDefinitionPathSet
var availableVersionGlobalMutex sync.Mutex

type meshmodelDefinitionPathSet struct {
	meshmodelDefinitionPath string
}

func RegisterMeshModelComponents(uuid, runtime, host, port string) error {
	meshmodelRDP := []adapter.MeshModelRegistrantDefinitionPath{}
	pathSets, err := loadMeshmodelComponents(MeshmodelComponents)
	if err != nil {
		return err
	}
	portint, _ := strconv.Atoi(port)
	for _, pathSet := range pathSets {
		meshmodelRDP = append(meshmodelRDP, adapter.MeshModelRegistrantDefinitionPath{
			EntityDefintionPath: pathSet.meshmodelDefinitionPath,
			Host:                host,
			Port:                portint,
			Type:                types.ComponentDefinition,
		})
	}

	return adapter.
		NewMeshModelRegistrant(meshmodelRDP, fmt.Sprintf("%s/api/meshmodel/components/register", runtime)).
		Register(uuid)
}

// RegisterWorkloads will register all of the workload definitions
// present in the path oam/workloads
//
// Registration process will send POST request to $runtime/api/oam/workload
func RegisterWorkloads(runtime, host string) error {
	oamRDP := []adapter.OAMRegistrantDefinitionPath{}

	for _, pathSet := range pathSets {
		metadata := map[string]string{
			config.OAMAdapterNameMetadataKey: config.ConsulOperation,
		}

		if strings.HasSuffix(pathSet.name, "addon") {
			metadata[config.OAMComponentCategoryMetadataKey] = "addon"
		}

		oamRDP = append(oamRDP, adapter.OAMRegistrantDefinitionPath{
			OAMDefintionPath: pathSet.oamDefinitionPath,
			OAMRefSchemaPath: pathSet.jsonSchemaPath,
			Host:             host,
			Metadata:         metadata,
		})
	}

	return adapter.
		NewOAMRegistrant(oamRDP, fmt.Sprintf("%s/api/oam/workload", runtime)).
		Register()
}

// RegisterTraits will register all of the trait definitions
// present in the path oam/traits
//
// Registeration process will send POST request to $runtime/api/oam/trait
func RegisterTraits(runtime, host string) error {
	oamRDP := []adapter.OAMRegistrantDefinitionPath{}

	pathSets, err := load(traitPath)
	if err != nil {
		return ErrLoadingPathset(err)
	}

	for _, pathSet := range pathSets {
		metadata := map[string]string{
			config.OAMAdapterNameMetadataKey: config.ConsulOperation,
		}

		oamRDP = append(oamRDP, adapter.OAMRegistrantDefinitionPath{
			OAMDefintionPath: pathSet.oamDefinitionPath,
			OAMRefSchemaPath: pathSet.jsonSchemaPath,
			Host:             host,
			Metadata:         metadata,
		})
	}

	return adapter.
		NewOAMRegistrant(oamRDP, fmt.Sprintf("%s/api/oam/trait", runtime)).
		Register()
}

func loadMeshmodelComponents(basepath string) ([]meshmodelDefinitionPathSet, error) {
	res := []meshmodelDefinitionPathSet{}
	if err := filepath.Walk(basepath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		res = append(res, meshmodelDefinitionPathSet{
			meshmodelDefinitionPath: path,
		})
		availableVersionGlobalMutex.Lock()
		AvailableVersions[filepath.Base(filepath.Dir(path))] = true // Getting available versions already existing on file system
		availableVersionGlobalMutex.Unlock()
		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

func load(basePath string) ([]schemaDefinitionPathSet, error) {
	res := []schemaDefinitionPathSet{}

	if err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if matched, err := filepath.Match("*_definition.json", filepath.Base(path)); err != nil {
			return err
		} else if matched {
			nameWithPath := strings.TrimSuffix(path, "_definition.json")

			res = append(res, schemaDefinitionPathSet{
				oamDefinitionPath: path,
				jsonSchemaPath:    fmt.Sprintf("%s.meshery.layer5io.schema.json", nameWithPath),
				name:              filepath.Base(nameWithPath),
			})
			availableVersionGlobalMutex.Lock()
			AvailableVersions[filepath.Base(filepath.Dir(path))] = true
			availableVersionGlobalMutex.Unlock()
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

func init() {
	var err error
	pathSets, err = load(WorkloadPath)
	if err != nil {
		fmt.Println("Error while loading components: ", err.Error())
	}
}
