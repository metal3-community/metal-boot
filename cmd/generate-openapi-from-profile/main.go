package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// InteropProfile represents the structure of a Redfish Interop Profile
type InteropProfile struct {
	ProfileName     string                      `json:"ProfileName"`
	ProfileVersion  string                      `json:"ProfileVersion"`
	Purpose         string                      `json:"Purpose"`
	OwningEntity    string                      `json:"OwningEntity"`
	ContactInfo     string                      `json:"ContactInfo"`
	License         string                      `json:"License"`
	Resources       map[string]ResourceProfile  `json:"Resources"`
	Registries      map[string]RegistryProfile  `json:"Registries,omitempty"`
	Protocol        ProtocolProfile             `json:"Protocol"`
}

// ResourceProfile defines requirements for a specific Redfish resource
type ResourceProfile struct {
	Purpose              string                          `json:"Purpose"`
	ReadRequirement      string                          `json:"ReadRequirement,omitempty"`
	WriteRequirement     string                          `json:"WriteRequirement,omitempty"`
	ConditionalRequirement string                        `json:"ConditionalRequirement,omitempty"`
	MinVersion           string                          `json:"MinVersion,omitempty"`
	CreateResource       bool                            `json:"CreateResource,omitempty"`
	URIs                 []string                        `json:"URIs,omitempty"`
	PropertyRequirements map[string]PropertyRequirement `json:"PropertyRequirements,omitempty"`
	ActionRequirements   map[string]ActionRequirement   `json:"ActionRequirements,omitempty"`
	MinCount             int                             `json:"MinCount,omitempty"`
}

// PropertyRequirement defines requirements for a specific property
type PropertyRequirement struct {
	Purpose               string                          `json:"Purpose,omitempty"`
	ReadRequirement       string                          `json:"ReadRequirement,omitempty"`
	WriteRequirement      string                          `json:"WriteRequirement,omitempty"`
	ConditionalRequirement string                         `json:"ConditionalRequirement,omitempty"`
	Comparison            string                          `json:"Comparison,omitempty"`
	Values                []string                        `json:"Values,omitempty"`
	MinCount              int                             `json:"MinCount,omitempty"`
	MinSupportValues      []string                        `json:"MinSupportValues,omitempty"`
	PropertyRequirements  map[string]PropertyRequirement `json:"PropertyRequirements,omitempty"`
}

// ActionRequirement defines requirements for a specific action
type ActionRequirement struct {
	Purpose         string                          `json:"Purpose"`
	ReadRequirement string                          `json:"ReadRequirement,omitempty"`
	Parameters      map[string]ParameterRequirement `json:"Parameters,omitempty"`
}

// ParameterRequirement defines requirements for action parameters
type ParameterRequirement struct {
	ReadRequirement     string   `json:"ReadRequirement,omitempty"`
	ParameterValues     []string `json:"ParameterValues,omitempty"`
	RecommendedValues   []string `json:"RecommendedValues,omitempty"`
}

// ProtocolProfile defines protocol requirements
type ProtocolProfile struct {
	MinVersion string `json:"MinVersion"`
}

// RegistryProfile defines registry requirements
type RegistryProfile struct {
	MinVersion string                 `json:"MinVersion"`
	Messages   map[string]interface{} `json:"Messages,omitempty"`
}

// OpenAPI specification structures
type OpenAPISpec struct {
	OpenAPI    string                 `yaml:"openapi"`
	Info       OpenAPIInfo            `yaml:"info"`
	Servers    []OpenAPIServer        `yaml:"servers,omitempty"`
	Paths      map[string]OpenAPIPath `yaml:"paths"`
	Components OpenAPIComponents      `yaml:"components"`
}

type OpenAPIInfo struct {
	Title       string            `yaml:"title"`
	Description string            `yaml:"description"`
	Version     string            `yaml:"version"`
	Contact     *OpenAPIContact   `yaml:"contact,omitempty"`
	License     *OpenAPILicense   `yaml:"license,omitempty"`
}

type OpenAPIContact struct {
	Name  string `yaml:"name,omitempty"`
	URL   string `yaml:"url,omitempty"`
	Email string `yaml:"email,omitempty"`
}

type OpenAPILicense struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url,omitempty"`
}

type OpenAPIServer struct {
	URL         string `yaml:"url"`
	Description string `yaml:"description,omitempty"`
}

type OpenAPIPath struct {
	Summary     string               `yaml:"summary,omitempty"`
	Description string               `yaml:"description,omitempty"`
	Parameters  []OpenAPIParameter   `yaml:"parameters,omitempty"`
	Get         *OpenAPIOperation    `yaml:"get,omitempty"`
	Post        *OpenAPIOperation    `yaml:"post,omitempty"`
	Patch       *OpenAPIOperation    `yaml:"patch,omitempty"`
	Put         *OpenAPIOperation    `yaml:"put,omitempty"`
	Delete      *OpenAPIOperation    `yaml:"delete,omitempty"`
}

type OpenAPIOperation struct {
	OperationID string                         `yaml:"operationId"`
	Summary     string                         `yaml:"summary,omitempty"`
	Description string                         `yaml:"description,omitempty"`
	Parameters  []OpenAPIParameter             `yaml:"parameters,omitempty"`
	RequestBody *OpenAPIRequestBody            `yaml:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse     `yaml:"responses"`
	Deprecated  bool                           `yaml:"deprecated,omitempty"`
}

type OpenAPIParameter struct {
	Name        string             `yaml:"name"`
	In          string             `yaml:"in"`
	Description string             `yaml:"description,omitempty"`
	Required    bool               `yaml:"required,omitempty"`
	Schema      OpenAPISchemaOrRef `yaml:"schema"`
}

type OpenAPIRequestBody struct {
	Description string                            `yaml:"description,omitempty"`
	Required    bool                              `yaml:"required,omitempty"`
	Content     map[string]OpenAPIMediaType       `yaml:"content"`
}

type OpenAPIResponse struct {
	Description string                            `yaml:"description"`
	Content     map[string]OpenAPIMediaType       `yaml:"content,omitempty"`
}

type OpenAPIMediaType struct {
	Schema OpenAPISchemaOrRef `yaml:"schema"`
}

type OpenAPIComponents struct {
	Schemas map[string]OpenAPISchema `yaml:"schemas"`
}

type OpenAPISchemaOrRef struct {
	Ref                  string                     `yaml:"$ref,omitempty"`
	Type                 string                     `yaml:"type,omitempty"`
	Format               string                     `yaml:"format,omitempty"`
	Description          string                     `yaml:"description,omitempty"`
	Properties           map[string]OpenAPISchemaOrRef `yaml:"properties,omitempty"`
	Required             []string                   `yaml:"required,omitempty"`
	Items                *OpenAPISchemaOrRef        `yaml:"items,omitempty"`
	Enum                 []interface{}              `yaml:"enum,omitempty"`
	ReadOnly             bool                       `yaml:"readOnly,omitempty"`
	WriteOnly            bool                       `yaml:"writeOnly,omitempty"`
	Nullable             bool                       `yaml:"nullable,omitempty"`
	Pattern              string                     `yaml:"pattern,omitempty"`
	Minimum              *float64                   `yaml:"minimum,omitempty"`
	Maximum              *float64                   `yaml:"maximum,omitempty"`
	MinLength            *int                       `yaml:"minLength,omitempty"`
	MaxLength            *int                       `yaml:"maxLength,omitempty"`
	AdditionalProperties interface{}                `yaml:"additionalProperties,omitempty"`
	AllOf                []OpenAPISchemaOrRef       `yaml:"allOf,omitempty"`
	OneOf                []OpenAPISchemaOrRef       `yaml:"oneOf,omitempty"`
	AnyOf                []OpenAPISchemaOrRef       `yaml:"anyOf,omitempty"`
}

type OpenAPISchema = OpenAPISchemaOrRef

// Generator handles the conversion from interop profile to OpenAPI
type Generator struct {
	profile     *InteropProfile
	openAPI     *OpenAPISpec
	pathMappings map[string]string
}

// New creates a new generator instance
func NewGenerator() *Generator {
	return &Generator{
		pathMappings: map[string]string{
			"ServiceRoot":               "/redfish/v1/",
			"ComputerSystemCollection":  "/redfish/v1/Systems/",
			"ComputerSystem":            "/redfish/v1/Systems/{systemId}",
			"ManagerCollection":         "/redfish/v1/Managers/",
			"Manager":                   "/redfish/v1/Managers/{managerId}",
			"ChassisCollection":         "/redfish/v1/Chassis/",
			"Chassis":                   "/redfish/v1/Chassis/{chassisId}",
			"EthernetInterfaceCollection": "/redfish/v1/Systems/{systemId}/EthernetInterfaces/",
			"EthernetInterface":         "/redfish/v1/Systems/{systemId}/EthernetInterfaces/{ethernetInterfaceId}",
			"Bios":                      "/redfish/v1/Systems/{systemId}/Bios",
			"SecureBoot":                "/redfish/v1/Systems/{systemId}/SecureBoot",
			"ProcessorCollection":       "/redfish/v1/Systems/{systemId}/Processors/",
			"Processor":                 "/redfish/v1/Systems/{systemId}/Processors/{processorId}",
			"SimpleStorageCollection":   "/redfish/v1/Systems/{systemId}/SimpleStorage/",
			"SimpleStorage":             "/redfish/v1/Systems/{systemId}/SimpleStorage/{simpleStorageId}",
			"StorageCollection":         "/redfish/v1/Systems/{systemId}/Storage/",
			"Storage":                   "/redfish/v1/Systems/{systemId}/Storage/{storageId}",
			"VolumeCollection":          "/redfish/v1/Systems/{systemId}/Storage/{storageId}/Volumes/",
			"Volume":                    "/redfish/v1/Systems/{systemId}/Storage/{storageId}/Volumes/{volumeId}",
			"DriveCollection":           "/redfish/v1/Systems/{systemId}/Storage/{storageId}/Drives/",
			"Drive":                     "/redfish/v1/Systems/{systemId}/Storage/{storageId}/Drives/{driveId}",
			"VirtualMediaCollection":    "/redfish/v1/Managers/{managerId}/VirtualMedia/",
			"VirtualMedia":              "/redfish/v1/Managers/{managerId}/VirtualMedia/{virtualMediaId}",
			"Power":                     "/redfish/v1/Chassis/{chassisId}/Power",
			"Thermal":                   "/redfish/v1/Chassis/{chassisId}/Thermal",
			"SessionService":            "/redfish/v1/SessionService",
			"TaskService":               "/redfish/v1/TaskService",
			"UpdateService":             "/redfish/v1/UpdateService",
		},
	}
}

// LoadProfile loads an interop profile from a URL or file
func (g *Generator) LoadProfile(source string) error {
	var data []byte
	var err error

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		resp, err := http.Get(source)
		if err != nil {
			return fmt.Errorf("failed to download profile: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download profile: HTTP %d", resp.StatusCode)
		}

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read profile: %w", err)
		}
	} else {
		data, err = os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("failed to read profile file: %w", err)
		}
	}

	g.profile = &InteropProfile{}
	if err := json.Unmarshal(data, g.profile); err != nil {
		return fmt.Errorf("failed to parse profile JSON: %w", err)
	}

	return nil
}

// Generate creates the OpenAPI specification
func (g *Generator) Generate() error {
	if g.profile == nil {
		return fmt.Errorf("no profile loaded")
	}

	g.openAPI = &OpenAPISpec{
		OpenAPI: "3.1.0",
		Info: OpenAPIInfo{
			Title:       fmt.Sprintf("Redfish API (%s)", g.profile.ProfileName),
			Description: fmt.Sprintf("Redfish OpenAPI specification compliant with %s v%s. %s", g.profile.ProfileName, g.profile.ProfileVersion, g.profile.Purpose),
			Version:     g.profile.ProfileVersion,
			Contact: &OpenAPIContact{
				Name:  g.profile.OwningEntity,
				Email: g.profile.ContactInfo,
			},
		},
		Servers: []OpenAPIServer{
			{URL: "/"},
		},
		Paths: make(map[string]OpenAPIPath),
		Components: OpenAPIComponents{
			Schemas: make(map[string]OpenAPISchema),
		},
	}

	// Add common schemas first
	g.addCommonSchemas()

	// Process each resource in the profile
	for resourceName, resourceProfile := range g.profile.Resources {
		if err := g.processResource(resourceName, resourceProfile); err != nil {
			return fmt.Errorf("failed to process resource %s: %w", resourceName, err)
		}
	}

	return nil
}

// addCommonSchemas adds standard Redfish schemas
func (g *Generator) addCommonSchemas() {
	// Redfish Error schema
	g.openAPI.Components.Schemas["RedfishError"] = OpenAPISchema{
		Type:        "object",
		Description: "Contains an error payload from a Redfish Service.",
		Properties: map[string]OpenAPISchemaOrRef{
			"error": {
				Type: "object",
				Properties: map[string]OpenAPISchemaOrRef{
					"code":    {Type: "string"},
					"message": {Type: "string"},
					"@Message.ExtendedInfo": {
						Type: "array",
						Items: &OpenAPISchemaOrRef{
							Ref: "#/components/schemas/Message",
						},
					},
				},
				Required: []string{"code", "message"},
			},
		},
		Required: []string{"error"},
	}

	// Message schema
	g.openAPI.Components.Schemas["Message"] = OpenAPISchema{
		Type: "object",
		Properties: map[string]OpenAPISchemaOrRef{
			"MessageId":   {Type: "string", ReadOnly: true},
			"Message":     {Type: "string", ReadOnly: true},
			"MessageArgs": {Type: "array", Items: &OpenAPISchemaOrRef{Type: "string"}, ReadOnly: true},
			"Severity":    {Type: "string", ReadOnly: true},
			"Resolution":  {Type: "string", ReadOnly: true},
		},
		Required: []string{"MessageId"},
	}

	// OData ID Reference
	g.openAPI.Components.Schemas["idRef"] = OpenAPISchema{
		Type: "object",
		Properties: map[string]OpenAPISchemaOrRef{
			"@odata.id": {Type: "string", Format: "uri-reference", ReadOnly: true},
		},
		Required: []string{"@odata.id"},
	}

	// Common enums and types
	commonEnums := map[string]OpenAPISchema{
		"Health": {
			Type: "string",
			Enum: []interface{}{"OK", "Warning", "Critical"},
		},
		"State": {
			Type: "string",
			Enum: []interface{}{"Enabled", "Disabled", "StandbyOffline", "StandbySpare", "InTest", "Starting", "Absent", "UnavailableOffline", "Deferring", "Quiesced", "Updating"},
		},
		"PowerState": {
			Type: "string",
			Enum: []interface{}{"On", "Off", "PoweringOn", "PoweringOff"},
		},
		"IndicatorLED": {
			Type: "string",
			Enum: []interface{}{"Unknown", "Lit", "Blinking", "Off"},
		},
		"Status": {
			Type: "object",
			Properties: map[string]OpenAPISchemaOrRef{
				"Health": {Ref: "#/components/schemas/Health"},
				"State":  {Ref: "#/components/schemas/State"},
			},
		},
	}

	for name, schema := range commonEnums {
		g.openAPI.Components.Schemas[name] = schema
	}
}

// processResource processes a single resource from the profile
func (g *Generator) processResource(resourceName string, resourceProfile ResourceProfile) error {
	// Skip resources that are not required or recommended
	if !g.isResourceRequired(resourceProfile) {
		return nil
	}

	// Get the path for this resource
	path, exists := g.pathMappings[resourceName]
	if !exists {
		// Skip resources without defined paths
		log.Printf("Warning: No path mapping for resource %s", resourceName)
		return nil
	}

	// Create the path operations
	pathItem := OpenAPIPath{}

	// Extract path parameters
	pathItem.Parameters = g.extractPathParameters(path)

	// Add GET operation for all resources
	pathItem.Get = g.createGetOperation(resourceName, resourceProfile)

	// Add other operations based on requirements
	if resourceProfile.CreateResource || g.hasWriteRequirement(resourceProfile, "POST") {
		if strings.Contains(resourceName, "Collection") {
			pathItem.Post = g.createPostOperation(resourceName, resourceProfile)
		}
	}

	if g.hasWriteRequirement(resourceProfile, "PATCH") {
		if !strings.Contains(resourceName, "Collection") {
			pathItem.Patch = g.createPatchOperation(resourceName, resourceProfile)
		}
	}

	// Add action operations
	for actionName, actionReq := range resourceProfile.ActionRequirements {
		if g.isActionRequired(actionReq) {
			actionPath := g.getActionPath(path, resourceName, actionName)
			actionPathItem := OpenAPIPath{
				Post: g.createActionOperation(resourceName, actionName, actionReq),
			}
			actionPathItem.Parameters = g.extractPathParameters(actionPath)
			g.openAPI.Paths[actionPath] = actionPathItem
		}
	}

	g.openAPI.Paths[path] = pathItem

	// Create schema for this resource
	g.createResourceSchema(resourceName, resourceProfile)

	return nil
}

// isResourceRequired checks if a resource is required or recommended
func (g *Generator) isResourceRequired(resource ResourceProfile) bool {
	requirement := strings.ToLower(resource.ReadRequirement)
	return requirement == "mandatory" || requirement == "recommended"
}

// hasWriteRequirement checks if a resource has write requirements
func (g *Generator) hasWriteRequirement(resource ResourceProfile, method string) bool {
	// Check if any properties have write requirements
	for _, prop := range resource.PropertyRequirements {
		if strings.ToLower(prop.WriteRequirement) == "mandatory" || strings.ToLower(prop.WriteRequirement) == "recommended" {
			return true
		}
	}
	return false
}

// isActionRequired checks if an action is required
func (g *Generator) isActionRequired(action ActionRequirement) bool {
	requirement := strings.ToLower(action.ReadRequirement)
	return requirement == "mandatory" || requirement == "recommended"
}

// extractPathParameters extracts parameters from a path template
func (g *Generator) extractPathParameters(path string) []OpenAPIParameter {
	var parameters []OpenAPIParameter
	
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(path, -1)
	
	for _, match := range matches {
		paramName := match[1]
		parameters = append(parameters, OpenAPIParameter{
			Name:        paramName,
			In:          "path",
			Description: fmt.Sprintf("ID of %s", strings.ToLower(strings.TrimSuffix(paramName, "Id"))),
			Required:    true,
			Schema:      OpenAPISchemaOrRef{Type: "string"},
		})
	}
	
	return parameters
}

// createGetOperation creates a GET operation for a resource
func (g *Generator) createGetOperation(resourceName string, resource ResourceProfile) *OpenAPIOperation {
	operationID := fmt.Sprintf("get_%s", strings.ToLower(resourceName))
	
	return &OpenAPIOperation{
		OperationID: operationID,
		Summary:     fmt.Sprintf("Get %s", resourceName),
		Description: resource.Purpose,
		Responses: map[string]OpenAPIResponse{
			"200": {
				Description: fmt.Sprintf("The %s resource", resourceName),
				Content: map[string]OpenAPIMediaType{
					"application/json": {
						Schema: OpenAPISchemaOrRef{
							Ref: fmt.Sprintf("#/components/schemas/%s", resourceName),
						},
					},
				},
			},
			"default": {
				Description: "Error condition",
				Content: map[string]OpenAPIMediaType{
					"application/json": {
						Schema: OpenAPISchemaOrRef{
							Ref: "#/components/schemas/RedfishError",
						},
					},
				},
			},
		},
	}
}

// createPostOperation creates a POST operation for collections
func (g *Generator) createPostOperation(resourceName string, resource ResourceProfile) *OpenAPIOperation {
	singularName := strings.TrimSuffix(resourceName, "Collection")
	operationID := fmt.Sprintf("create_%s", strings.ToLower(singularName))
	
	return &OpenAPIOperation{
		OperationID: operationID,
		Summary:     fmt.Sprintf("Create %s", singularName),
		Description: fmt.Sprintf("Create a new %s resource", singularName),
		RequestBody: &OpenAPIRequestBody{
			Required: true,
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: OpenAPISchemaOrRef{
						Ref: fmt.Sprintf("#/components/schemas/%s", singularName),
					},
				},
			},
		},
		Responses: map[string]OpenAPIResponse{
			"201": {
				Description: fmt.Sprintf("%s created successfully", singularName),
				Content: map[string]OpenAPIMediaType{
					"application/json": {
						Schema: OpenAPISchemaOrRef{
							Ref: fmt.Sprintf("#/components/schemas/%s", singularName),
						},
					},
				},
			},
			"default": {
				Description: "Error condition",
				Content: map[string]OpenAPIMediaType{
					"application/json": {
						Schema: OpenAPISchemaOrRef{
							Ref: "#/components/schemas/RedfishError",
						},
					},
				},
			},
		},
	}
}

// createPatchOperation creates a PATCH operation for resources
func (g *Generator) createPatchOperation(resourceName string, resource ResourceProfile) *OpenAPIOperation {
	operationID := fmt.Sprintf("patch_%s", strings.ToLower(resourceName))
	
	return &OpenAPIOperation{
		OperationID: operationID,
		Summary:     fmt.Sprintf("Update %s", resourceName),
		Description: fmt.Sprintf("Update properties of the %s resource", resourceName),
		RequestBody: &OpenAPIRequestBody{
			Required: true,
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: OpenAPISchemaOrRef{
						Ref: fmt.Sprintf("#/components/schemas/%s", resourceName),
					},
				},
			},
		},
		Responses: map[string]OpenAPIResponse{
			"200": {
				Description: fmt.Sprintf("%s updated successfully", resourceName),
				Content: map[string]OpenAPIMediaType{
					"application/json": {
						Schema: OpenAPISchemaOrRef{
							Ref: fmt.Sprintf("#/components/schemas/%s", resourceName),
						},
					},
				},
			},
			"204": {
				Description: "Success, but no response data",
			},
			"default": {
				Description: "Error condition",
				Content: map[string]OpenAPIMediaType{
					"application/json": {
						Schema: OpenAPISchemaOrRef{
							Ref: "#/components/schemas/RedfishError",
						},
					},
				},
			},
		},
	}
}

// getActionPath constructs the action path
func (g *Generator) getActionPath(basePath, resourceName, actionName string) string {
	// Remove trailing parameters for action paths
	basePath = strings.TrimSuffix(basePath, "/")
	return fmt.Sprintf("%s/Actions/%s.%s", basePath, resourceName, actionName)
}

// createActionOperation creates a POST operation for actions
func (g *Generator) createActionOperation(resourceName, actionName string, action ActionRequirement) *OpenAPIOperation {
	operationID := fmt.Sprintf("%s_%s", strings.ToLower(actionName), strings.ToLower(resourceName))
	
	requestSchema := OpenAPISchemaOrRef{
		Type: "object",
		Properties: make(map[string]OpenAPISchemaOrRef),
	}
	
	var required []string
	
	// Add parameters from action requirements
	for paramName, paramReq := range action.Parameters {
		paramSchema := g.createParameterSchema(paramReq)
		requestSchema.Properties[paramName] = paramSchema
		
		if strings.ToLower(paramReq.ReadRequirement) == "mandatory" {
			required = append(required, paramName)
		}
	}
	
	if len(required) > 0 {
		sort.Strings(required)
		requestSchema.Required = required
	}
	
	return &OpenAPIOperation{
		OperationID: operationID,
		Summary:     fmt.Sprintf("Perform %s action", actionName),
		Description: action.Purpose,
		RequestBody: &OpenAPIRequestBody{
			Required: true,
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: requestSchema,
				},
			},
		},
		Responses: map[string]OpenAPIResponse{
			"200": {
				Description: "Action completed successfully",
				Content: map[string]OpenAPIMediaType{
					"application/json": {
						Schema: OpenAPISchemaOrRef{
							Ref: "#/components/schemas/RedfishError",
						},
					},
				},
			},
			"204": {
				Description: "Success, but no response data",
			},
			"default": {
				Description: "Error condition",
				Content: map[string]OpenAPIMediaType{
					"application/json": {
						Schema: OpenAPISchemaOrRef{
							Ref: "#/components/schemas/RedfishError",
						},
					},
				},
			},
		},
	}
}

// createParameterSchema creates a schema for action parameters
func (g *Generator) createParameterSchema(param ParameterRequirement) OpenAPISchemaOrRef {
	schema := OpenAPISchemaOrRef{Type: "string"}
	
	if len(param.ParameterValues) > 0 {
		schema.Enum = make([]interface{}, len(param.ParameterValues))
		for i, v := range param.ParameterValues {
			schema.Enum[i] = v
		}
	} else if len(param.RecommendedValues) > 0 {
		schema.Enum = make([]interface{}, len(param.RecommendedValues))
		for i, v := range param.RecommendedValues {
			schema.Enum[i] = v
		}
	}
	
	return schema
}

// createResourceSchema creates a schema definition for a resource
func (g *Generator) createResourceSchema(resourceName string, resource ResourceProfile) {
	schema := OpenAPISchema{
		Type: "object",
		Properties: map[string]OpenAPISchemaOrRef{
			"@odata.context": {Type: "string", ReadOnly: true},
			"@odata.id":      {Type: "string", ReadOnly: true},
			"@odata.type":    {Type: "string", ReadOnly: true},
			"Id":             {Type: "string", ReadOnly: true},
			"Name":           {Type: "string", ReadOnly: true},
		},
		Required: []string{"@odata.id", "@odata.type", "Id", "Name"},
	}
	
	// Add resource-specific properties from profile requirements
	for propName, propReq := range resource.PropertyRequirements {
		propSchema := g.createPropertySchema(propName, propReq)
		schema.Properties[propName] = propSchema
		
		// Add to required if mandatory
		if strings.ToLower(propReq.ReadRequirement) == "mandatory" || strings.ToLower(propReq.WriteRequirement) == "mandatory" {
			schema.Required = append(schema.Required, propName)
		}
	}
	
	// Handle collection-specific properties
	if strings.Contains(resourceName, "Collection") {
		schema.Properties["Description"] = OpenAPISchemaOrRef{Type: "string", ReadOnly: true}
		schema.Properties["Members@odata.count"] = OpenAPISchemaOrRef{Type: "integer", ReadOnly: true}
		schema.Properties["Members"] = OpenAPISchemaOrRef{
			Type: "array",
			Items: &OpenAPISchemaOrRef{
				Ref: "#/components/schemas/idRef",
			},
			ReadOnly: true,
		}
		schema.Required = append(schema.Required, "Members")
	}
	
	// Sort required properties for consistency
	if len(schema.Required) > 0 {
		sort.Strings(schema.Required)
	}
	
	g.openAPI.Components.Schemas[resourceName] = schema
}

// createPropertySchema creates a schema for a property based on profile requirements
func (g *Generator) createPropertySchema(propName string, propReq PropertyRequirement) OpenAPISchemaOrRef {
	// Handle special cases and complex properties
	switch propName {
	case "Status":
		return OpenAPISchemaOrRef{Ref: "#/components/schemas/Status"}
	case "PowerState":
		return OpenAPISchemaOrRef{Ref: "#/components/schemas/PowerState"}
	case "IndicatorLED":
		return OpenAPISchemaOrRef{Ref: "#/components/schemas/IndicatorLED"}
	case "MACAddress":
		return OpenAPISchemaOrRef{
			Type:    "string",
			Pattern: "^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$",
		}
	case "UUID":
		return OpenAPISchemaOrRef{
			Type:    "string",
			Pattern: "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$",
		}
	}
	
	// Handle link properties
	if propReq.Comparison == "LinkToResource" && len(propReq.Values) > 0 {
		return OpenAPISchemaOrRef{Ref: "#/components/schemas/idRef"}
	}
	
	// Handle enum properties
	if len(propReq.Values) > 0 {
		schema := OpenAPISchemaOrRef{Type: "string"}
		schema.Enum = make([]interface{}, len(propReq.Values))
		for i, v := range propReq.Values {
			schema.Enum[i] = v
		}
		return schema
	}
	
	// Handle nested properties
	if len(propReq.PropertyRequirements) > 0 {
		schema := OpenAPISchemaOrRef{
			Type:       "object",
			Properties: make(map[string]OpenAPISchemaOrRef),
		}
		
		var required []string
		for nestedPropName, nestedPropReq := range propReq.PropertyRequirements {
			schema.Properties[nestedPropName] = g.createPropertySchema(nestedPropName, nestedPropReq)
			if strings.ToLower(nestedPropReq.ReadRequirement) == "mandatory" {
				required = append(required, nestedPropName)
			}
		}
		
		if len(required) > 0 {
			sort.Strings(required)
			schema.Required = required
		}
		
		return schema
	}
	
	// Handle arrays
	if propReq.MinCount > 0 {
		return OpenAPISchemaOrRef{
			Type: "array",
			Items: &OpenAPISchemaOrRef{
				Ref: "#/components/schemas/idRef",
			},
		}
	}
	
	// Default to string for simple properties
	schema := OpenAPISchemaOrRef{Type: "string"}
	
	// Set read-only for read-only properties
	if strings.ToLower(propReq.WriteRequirement) == "" && strings.ToLower(propReq.ReadRequirement) != "" {
		schema.ReadOnly = true
	}
	
	return schema
}

// WriteSpec writes the OpenAPI specification to a file
func (g *Generator) WriteSpec(filename string) error {
	if g.openAPI == nil {
		return fmt.Errorf("no OpenAPI specification generated")
	}

	// Create output directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(g.openAPI)
	if err != nil {
		return fmt.Errorf("failed to marshal OpenAPI spec: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func main() {
	var (
		profileURL = flag.String("profile-url", "https://raw.githubusercontent.com/openstack/ironic/refs/heads/master/redfish-interop-profiles/OpenStackIronicProfile.v1_1_0.json", "URL or path to the Redfish Interop Profile JSON file")
		outputFile = flag.String("output", "api/redfish/openapi-from-profile.yaml", "Output file for the OpenAPI specification")
		help       = flag.Bool("help", false, "Show help message")
	)
	flag.Parse()

	if *help {
		fmt.Println("Redfish Interop Profile to OpenAPI Generator")
		fmt.Println("=" + strings.Repeat("=", 49))
		fmt.Println()
		fmt.Println("Generates a strict OpenAPI specification from a Redfish Interop Profile.")
		fmt.Println("Only includes resources and properties explicitly defined in the profile.")
		fmt.Println()
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Generate from default OpenStack Ironic profile")
		fmt.Println("  go run cmd/generate-openapi-from-profile/main.go")
		fmt.Println()
		fmt.Println("  # Generate from custom profile")
		fmt.Println("  go run cmd/generate-openapi-from-profile/main.go \\")
		fmt.Println("    -profile-url=https://example.com/profile.json \\")
		fmt.Println("    -output=custom-openapi.yaml")
		fmt.Println()
		fmt.Println("  # Generate from local file")
		fmt.Println("  go run cmd/generate-openapi-from-profile/main.go \\")
		fmt.Println("    -profile-url=./my-profile.json \\")
		fmt.Println("    -output=output.yaml")
		return
	}

	fmt.Println("Redfish Interop Profile to OpenAPI Generator")
	fmt.Println("=" + strings.Repeat("=", 49))
	fmt.Printf("Profile source: %s\n", *profileURL)
	fmt.Printf("Output file: %s\n", *outputFile)
	fmt.Println()

	// Create generator
	generator := NewGenerator()

	// Load profile
	fmt.Print("Loading interop profile... ")
	if err := generator.LoadProfile(*profileURL); err != nil {
		fmt.Printf("FAILED\n")
		log.Fatalf("Error loading profile: %v", err)
	}
	fmt.Printf("OK\n")
	fmt.Printf("  Profile: %s v%s\n", generator.profile.ProfileName, generator.profile.ProfileVersion)
	fmt.Printf("  Resources: %d\n", len(generator.profile.Resources))

	// Generate OpenAPI specification
	fmt.Print("Generating OpenAPI specification... ")
	if err := generator.Generate(); err != nil {
		fmt.Printf("FAILED\n")
		log.Fatalf("Error generating specification: %v", err)
	}
	fmt.Printf("OK\n")
	fmt.Printf("  Paths: %d\n", len(generator.openAPI.Paths))
	fmt.Printf("  Schemas: %d\n", len(generator.openAPI.Components.Schemas))

	// Write specification
	fmt.Printf("Writing specification to %s... ", *outputFile)
	if err := generator.WriteSpec(*outputFile); err != nil {
		fmt.Printf("FAILED\n")
		log.Fatalf("Error writing specification: %v", err)
	}
	fmt.Printf("OK\n")

	fmt.Println()
	fmt.Println("✓ Generation complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Review the generated specification")
	fmt.Println("2. Validate using tools like:")
	fmt.Println("   - swagger-codegen validate -i", *outputFile)
	fmt.Println("   - DMTF Redfish Interop Validator")
	fmt.Println("3. Update server implementation to match the spec")
	fmt.Println("4. Test with Redfish conformance tools")
}