# OpenAPI Generator from Redfish Interop Profiles

This Go tool generates strict OpenAPI 3.1.0 specifications from Redfish Interoperability Profiles, ensuring **mutually exclusive** compliance - only resources and properties explicitly defined in the interop profile are included.

## Key Features

- **Strict Compliance**: Only includes resources/properties explicitly required or recommended in the interop profile
- **No Assumptions**: Does not add common Redfish patterns unless specified in the profile
- **Native Go**: Leverages Go's type system for robust profile parsing
- **OpenAPI 3.1.0**: Generates modern OpenAPI specifications
- **Action Support**: Includes profile-defined actions with parameter validation

## Usage

### Basic Usage

```bash
# Generate from default OpenStack Ironic profile
go run cmd/generate-openapi-from-profile/main.go

# Output to specific file
go run cmd/generate-openapi-from-profile/main.go \
  -output=api/redfish/openapi-strict.yaml
```

### Custom Profiles

```bash
# Use custom profile URL
go run cmd/generate-openapi-from-profile/main.go \
  -profile-url=https://example.com/my-profile.json \
  -output=custom-openapi.yaml

# Use local profile file
go run cmd/generate-openapi-from-profile/main.go \
  -profile-url=./profiles/my-profile.json \
  -output=local-openapi.yaml
```

### Command Line Options

- `-profile-url`: URL or path to the Redfish Interop Profile JSON file
  - Default: OpenStack Ironic Profile v1.1.0 from GitHub
- `-output`: Output file for the OpenAPI specification
  - Default: `api/redfish/openapi-from-profile.yaml`
- `-help`: Show usage information

## What Gets Generated

### Resources Included
- Only resources with `ReadRequirement: "Mandatory"` or `ReadRequirement: "Recommended"`
- Resources must have defined path mappings in the tool

### Properties Included
- Only properties explicitly listed in `PropertyRequirements`
- Properties maintain read/write requirements from the profile
- Nested properties follow the same strict inclusion rules

### Operations Generated
- **GET**: All included resources
- **POST**: Collections with `CreateResource: true` or write requirements
- **PATCH**: Resources with property write requirements
- **Actions**: Only actions with `ReadRequirement: "Mandatory"` or `ReadRequirement: "Recommended"`

### Schemas Generated
- Minimal schemas containing only profile-specified properties
- Standard Redfish metadata properties (`@odata.id`, `@odata.type`, etc.)
- Proper OpenAPI references for linked resources
- Enum constraints from profile values

## Profile Structure Support

The tool supports the standard Redfish Interoperability Profile format:

```json
{
  "ProfileName": "ExampleProfile",
  "ProfileVersion": "1.0.0",
  "Resources": {
    "ResourceName": {
      "ReadRequirement": "Mandatory|Recommended|IfImplemented",
      "PropertyRequirements": {
        "PropertyName": {
          "ReadRequirement": "Mandatory|Recommended",
          "WriteRequirement": "Mandatory|Recommended",
          "Values": ["enum", "values"],
          "Comparison": "LinkToResource|AnyOf",
          "PropertyRequirements": { /* nested */ }
        }
      },
      "ActionRequirements": {
        "ActionName": {
          "ReadRequirement": "Mandatory|Recommended",
          "Parameters": {
            "ParamName": {
              "ParameterValues": ["value1", "value2"]
            }
          }
        }
      }
    }
  }
}
```

## Example Output

The generated OpenAPI specification will be structured like:

```yaml
openapi: 3.1.0
info:
  title: "Redfish API (ProfileName)"
  version: "1.0.0"
paths:
  /redfish/v1/Systems/{systemId}:
    get:
      operationId: get_computersystem
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ComputerSystem'
    patch:
      # Only if write requirements exist
components:
  schemas:
    ComputerSystem:
      type: object
      properties:
        # Only properties from PropertyRequirements
      required:
        # Only mandatory properties
```

## Integration with metal-boot

To use with the metal-boot project:

1. **Generate specification**:
   ```bash
   go run cmd/generate-openapi-from-profile/main.go \
     -output=api/redfish/openapi-interop.yaml
   ```

2. **Compare with existing**:
   ```bash
   diff api/redfish/openapi.yaml api/redfish/openapi-interop.yaml
   ```

3. **Update server implementation**:
   - Review `api/redfish/server.go`
   - Add/remove handlers as needed
   - Update response schemas

4. **Test compliance**:
   ```bash
   # Validate OpenAPI syntax
   swagger-codegen validate -i api/redfish/openapi-interop.yaml
   
   # Test with interop validators
   redfish-interop-validator ...
   ```

## Validation

After generation, validate the specification:

```bash
# Syntax validation
swagger-codegen validate -i output.yaml

# Redfish compliance (if tools available)
redfish-interop-validator -p profile.json -s output.yaml

# Online validation
# Upload to https://editor.swagger.io/
```

## Troubleshooting

### Missing Resources
If expected resources are missing:
1. Check if the resource has `ReadRequirement: "Mandatory"` or `ReadRequirement: "Recommended"`
2. Verify the resource name exists in the path mappings
3. Ensure the profile JSON is valid

### Missing Properties  
If properties are missing:
1. Check if the property is listed in `PropertyRequirements`
2. Verify the property has read or write requirements
3. Check for typos in property names

### Missing Actions
If actions are missing:
1. Verify `ActionRequirements` exists for the resource
2. Check if the action has `ReadRequirement: "Mandatory"` or `ReadRequirement: "Recommended"`
3. Ensure action names match expected patterns

## Development

### Adding New Resource Types

To support new resource types, update the `pathMappings` in `main.go`:

```go
pathMappings := map[string]string{
    "NewResource": "/redfish/v1/New/{newId}",
    "NewResourceCollection": "/redfish/v1/New/",
}
```

### Custom Property Handling

Add special case handling in `createPropertySchema()`:

```go
switch propName {
case "CustomProperty":
    return OpenAPISchemaOrRef{
        Type: "string",
        Pattern: "^custom-pattern$",
    }
}
```

### Debugging

Add debug output by modifying the log statements:

```go
log.Printf("Processing resource: %s", resourceName)
log.Printf("  ReadRequirement: %s", resourceProfile.ReadRequirement)
log.Printf("  Properties: %d", len(resourceProfile.PropertyRequirements))
```

## Architecture

The tool follows this processing flow:

1. **Load Profile**: Download or read interop profile JSON
2. **Parse Structure**: Unmarshal into Go structs with validation
3. **Filter Resources**: Include only mandatory/recommended resources
4. **Generate Paths**: Create OpenAPI paths with appropriate operations
5. **Create Schemas**: Build minimal schemas from profile requirements
6. **Output YAML**: Marshal to OpenAPI 3.1.0 YAML format

The design emphasizes **strict compliance** over completeness, ensuring the generated specification exactly matches the interoperability requirements without additional assumptions.