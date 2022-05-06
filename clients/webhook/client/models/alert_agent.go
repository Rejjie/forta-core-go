// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// AlertAgent alert agent
//
// swagger:model AlertAgent
type AlertAgent struct {

	// id
	// Example: 0x17381ae942ee1fe141d0652e9dad7d001761552f906fb1684b2812603de31049
	ID string `json:"id,omitempty"`

	// Docker image reference (Disco)
	// Example: bafybeibrigevnhic4befnkqbaagzgxqtdyv2fdgcbqwxe7ees3hw6fymme@sha256:9ca1547e130a6264bb1b4ad6b10f17cabf404957f23d457a30046b9afdf29fc8
	Image string `json:"image,omitempty"`

	// Agent reference (IPFS hash)
	// Example: QmU6L9Zo5rweF6QZLhLfwAAFUFRMF3uFdSnMiJzENXr37R
	Reference string `json:"reference,omitempty"`
}

// Validate validates this alert agent
func (m *AlertAgent) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this alert agent based on context it is used
func (m *AlertAgent) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *AlertAgent) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *AlertAgent) UnmarshalBinary(b []byte) error {
	var res AlertAgent
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}