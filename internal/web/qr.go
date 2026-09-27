package web

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	qrservice "github.com/jayzone91/media-converter/internal/qr"
)

const maxQRLogoSize = 2 * 1024 * 1024

type qrGenerateRequest struct {
	Type string `json:"type"`

	URL   string `json:"url"`
	Text  string `json:"text"`
	Phone string `json:"phone"`

	WiFi qrWiFiRequest `json:"wifi"`

	VCard qrVCardRequest `json:"vcard"`

	Event qrEventRequest `json:"event"`

	Style qrStyleRequest `json:"style"`
}

type qrWiFiRequest struct {
	SSID       string `json:"ssid"`
	Password   string `json:"password"`
	Encryption string `json:"encryption"`
	Hidden     bool   `json:"hidden"`
}

type qrVCardRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`

	Company  string `json:"company"`
	Position string `json:"position"`

	PhoneWork  string `json:"phone_work"`
	PhoneHome  string `json:"phone_home"`
	MobileWork string `json:"mobile_work"`
	MobileHome string `json:"mobile_home"`
	FaxWork    string `json:"fax_work"`

	Email   string `json:"email"`
	Website string `json:"website"`

	Street     string `json:"street"`
	PostalCode string `json:"postal_code"`
	City       string `json:"city"`
	Region     string `json:"region"`
	Country    string `json:"country"`
}

type qrEventRequest struct {
	Title string `json:"title"`

	Start string `json:"start"`
	End   string `json:"end"`

	Location    string `json:"location"`
	Description string `json:"description"`
}

type qrStyleRequest struct {
	Foreground string `json:"foreground"`
	Background string `json:"background"`

	GradientEnabled bool   `json:"gradient_enabled"`
	GradientStart   string `json:"gradient_start"`
	GradientEnd     string `json:"gradient_end"`

	Module      string `json:"module"`
	CornerOuter string `json:"corner_outer"`
	CornerInner string `json:"corner_inner"`

	HasLogo bool   `json:"has_logo"`
	Logo    string `json:"logo"`
}

type qrGenerateResponse struct {
	SVG string `json:"svg"`

	Version int `json:"version"`

	ErrorCorrection string `json:"error_correction"`
}

func (s *Server) handleQRGenerate(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request qrGenerateRequest

	decoder :=
		json.NewDecoder(
			r.Body,
		)

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		&request,
	); err != nil {
		http.Error(
			w,
			"Ungültige QR-Code-Anfrage.",
			http.StatusBadRequest,
		)

		return
	}

	payload, err :=
		buildQRPayload(
			request,
		)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)

		return
	}

	style, err :=
		buildQRStyle(
			request.Style,
		)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)

		return
	}

	matrix, err :=
		qrservice.Generate(
			payload,
			style,
		)
	if err != nil {
		http.Error(
			w,
			"QR Code konnte nicht erzeugt werden.",
			http.StatusInternalServerError,
		)

		return
	}

	svg, err :=
		qrservice.RenderSVG(
			matrix,
			style,
		)
	if err != nil {
		http.Error(
			w,
			"QR Code konnte nicht gerendert werden.",
			http.StatusInternalServerError,
		)

		return
	}

	response :=
		qrGenerateResponse{
			SVG: string(svg),

			Version: matrix.Version,

			ErrorCorrection: string(
				matrix.ErrorCorrection,
			),
		}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	_ = json.NewEncoder(
		w,
	).Encode(
		response,
	)
}

func buildQRPayload(
	request qrGenerateRequest,
) (string, error) {
	switch strings.ToLower(
		strings.TrimSpace(
			request.Type,
		),
	) {
	case "url":
		return qrservice.URL(
			request.URL,
		)

	case "text":
		return qrservice.Text(
			request.Text,
		)

	case "phone":
		return qrservice.Phone(
			request.Phone,
		)

	case "wifi":
		return qrservice.WiFiPayload(
			qrservice.WiFi{
				SSID: request.WiFi.SSID,

				Password: request.WiFi.Password,

				Encryption: request.WiFi.Encryption,

				Hidden: request.WiFi.Hidden,
			},
		)

	case "vcard":
		return qrservice.VCardPayload(
			qrservice.VCard{
				FirstName: request.VCard.FirstName,

				LastName: request.VCard.LastName,

				Company: request.VCard.Company,

				Position: request.VCard.Position,

				PhoneWork: request.VCard.PhoneWork,

				PhoneHome: request.VCard.PhoneHome,

				MobileWork: request.VCard.MobileWork,

				MobileHome: request.VCard.MobileHome,

				FaxWork: request.VCard.FaxWork,

				Email: request.VCard.Email,

				Website: request.VCard.Website,

				Street: request.VCard.Street,

				PostalCode: request.VCard.PostalCode,

				City: request.VCard.City,

				Region: request.VCard.Region,

				Country: request.VCard.Country,
			},
		)

	case "event":
		start, err :=
			parseQRDateTime(
				request.Event.Start,
			)
		if err != nil {
			return "", fmt.Errorf(
				"Ungültiger Beginn",
			)
		}

		end, err :=
			parseQRDateTime(
				request.Event.End,
			)
		if err != nil {
			return "", fmt.Errorf(
				"Ungültiges Ende",
			)
		}

		return qrservice.EventPayload(
			qrservice.Event{
				Title: request.Event.Title,

				Start: start,

				End: end,

				Location: request.Event.Location,

				Description: request.Event.Description,
			},
		)

	default:
		return "", fmt.Errorf(
			"Unbekannter QR-Code-Typ",
		)
	}
}

func buildQRStyle(
	request qrStyleRequest,
) (qrservice.Style, error) {
	style :=
		qrservice.DefaultStyle()

	if request.Foreground != "" {
		style.Foreground =
			request.Foreground
	}

	if request.Background != "" {
		style.Background =
			request.Background
	}

	style.Gradient =
		qrservice.Gradient{
			Enabled: request.GradientEnabled,

			Start: request.GradientStart,

			End: request.GradientEnd,
		}

	switch request.Module {
	case "", "square":
		style.Module =
			qrservice.ModuleSquare

	case "rounded":
		style.Module =
			qrservice.ModuleRounded

	case "dots":
		style.Module =
			qrservice.ModuleDots

	case "diamond":
		style.Module =
			qrservice.ModuleDiamond

	case "classy":
		style.Module =
			qrservice.ModuleClassy

	default:
		return style, fmt.Errorf(
			"Ungültiger QR-Code-Stil",
		)
	}

	switch request.CornerOuter {
	case "", "square":
		style.CornerOuter =
			qrservice.CornerOuterSquare

	case "rounded":
		style.CornerOuter =
			qrservice.CornerOuterRounded

	case "extra-rounded":
		style.CornerOuter =
			qrservice.CornerOuterExtraRounded

	default:
		return style, fmt.Errorf(
			"Ungültiger äußerer Eckenstil",
		)
	}

	switch request.CornerInner {
	case "", "square":
		style.CornerInner =
			qrservice.CornerInnerSquare

	case "dot":
		style.CornerInner =
			qrservice.CornerInnerDot

	case "rounded":
		style.CornerInner =
			qrservice.CornerInnerRounded

	default:
		return style, fmt.Errorf(
			"Ungültiger innerer Eckenstil",
		)
	}

	if style.Gradient.Enabled {
		if style.Gradient.Start == "" {
			style.Gradient.Start =
				style.Foreground
		}

		if style.Gradient.End == "" {
			style.Gradient.End =
				style.Foreground
		}
	}

	if request.HasLogo {
		logo, err :=
			validateQRLogo(
				request.Logo,
			)
		if err != nil {
			return style, err
		}

		style.HasLogo =
			true

		style.Logo =
			logo
	}

	return style, nil
}

func validateQRLogo(
	value string,
) (string, error) {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return "", fmt.Errorf(
			"Logo fehlt",
		)
	}

	var prefix string

	switch {
	case strings.HasPrefix(
		value,
		"data:image/png;base64,",
	):
		prefix =
			"data:image/png;base64,"

	case strings.HasPrefix(
		value,
		"data:image/jpeg;base64,",
	):
		prefix =
			"data:image/jpeg;base64,"

	case strings.HasPrefix(
		value,
		"data:image/webp;base64,",
	):
		prefix =
			"data:image/webp;base64,"

	default:
		return "", fmt.Errorf(
			"Logo muss PNG, JPEG oder WEBP sein",
		)
	}

	encoded :=
		strings.TrimPrefix(
			value,
			prefix,
		)

	data, err :=
		base64.StdEncoding.DecodeString(
			encoded,
		)
	if err != nil {
		return "", fmt.Errorf(
			"Logo ist ungültig",
		)
	}

	if len(data) == 0 {
		return "", fmt.Errorf(
			"Logo ist leer",
		)
	}

	if len(data) >
		maxQRLogoSize {
		return "", fmt.Errorf(
			"Logo darf maximal 2 MiB groß sein",
		)
	}

	return value, nil
}

func parseQRDateTime(
	value string,
) (time.Time, error) {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return time.Time{},
			fmt.Errorf(
				"date is empty",
			)
	}

	return time.ParseInLocation(
		"2006-01-02T15:04",
		value,
		time.Local,
	)
}
