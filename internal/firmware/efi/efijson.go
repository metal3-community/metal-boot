package efi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// JSONEncoder handles serializing EFI data types to JSON
type JSONEncoder struct{}

// EfiVarJSON represents the JSON structure for an EFI variable
type EfiVarJSON struct {
	Name string `json:"name"`
	GUID string `json:"guid"`
	Attr int    `json:"attr"`
	Data string `json:"data"`           // hex encoded
	Time string `json:"time,omitempty"` // hex encoded
}

// EfiVarListJSON represents the JSON structure for a list of EFI variables
type EfiVarListJSON struct {
	Version   int          `json:"version"`
	Variables []EfiVarJSON `json:"variables"`
}

// MarshalEfiVar converts an EfiVar to its JSON representation
func (e *JSONEncoder) MarshalEfiVar(v *EfiVar) EfiVarJSON {
	result := EfiVarJSON{
		Name: v.Name.String(),
		GUID: v.Guid.String(),
		Attr: int(v.Attr),
		Data: hex.EncodeToString(v.Data),
	}

	if v.Time != nil {
		result.Time = hex.EncodeToString(v.BytesTime())
	}

	return result
}

// MarshalEfiVarList converts an EfiVarList to its JSON representation
func (e *JSONEncoder) MarshalEfiVarList(list EfiVarList) EfiVarListJSON {
	variables := make([]EfiVarJSON, 0, len(list))

	for _, item := range list {
		variables = append(variables, e.MarshalEfiVar(item))
	}

	return EfiVarListJSON{
		Version:   2,
		Variables: variables,
	}
}

// MarshalJSON implements the json.Marshaler interface for EfiVar
func (v *EfiVar) MarshalJSON() ([]byte, error) {
	encoder := JSONEncoder{}
	return json.Marshal(encoder.MarshalEfiVar(v))
}

// MarshalJSON implements the json.Marshaler interface for EfiVarList
func (list EfiVarList) MarshalJSON() ([]byte, error) {
	encoder := JSONEncoder{}
	return json.Marshal(encoder.MarshalEfiVarList(list))
}

// UnmarshalJSON implements the json.Unmarshaler interface for EfiVar
func (v *EfiVar) UnmarshalJSON(data []byte) error {
	var jsonVar EfiVarJSON
	if err := json.Unmarshal(data, &jsonVar); err != nil {
		return err
	}

	name := FromString(jsonVar.Name)

	guid, err := GUIDFromBytes([]byte(jsonVar.GUID)) // ParseGUIDString(jsonVar.GUID)
	if err != nil {
		return err
	}

	varData, err := hex.DecodeString(jsonVar.Data)
	if err != nil {
		return err
	}

	v.Name = name
	v.Guid = guid
	v.Attr = uint32(jsonVar.Attr)
	v.Data = varData

	if jsonVar.Time != "" {
		timeData, err := hex.DecodeString(jsonVar.Time)
		if err != nil {
			return err
		}
		if err := v.ParseTime(timeData, 0); err != nil {
			return err
		}
	}

	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for EfiVarList
func (list *EfiVarList) UnmarshalJSON(data []byte) error {
	var jsonList struct {
		Version   int               `json:"version"`
		Variables []json.RawMessage `json:"variables"`
	}

	if err := json.Unmarshal(data, &jsonList); err != nil {
		return err
	}

	if jsonList.Version != 2 {
		return fmt.Errorf("unsupported EfiVarList version: %d", jsonList.Version)
	}

	*list = make(EfiVarList)

	for _, varData := range jsonList.Variables {
		var v EfiVar
		if err := json.Unmarshal(varData, &v); err != nil {
			return err
		}
		(*list)[string(v.Name.String())] = &v
	}

	return nil
}

// Custom JSON decoder function for use with json.Unmarshal
func DecodeEfiJSON(data []byte, v *EfiVarJSON) error {
	return json.Unmarshal(data, v)
}
