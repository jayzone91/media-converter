package web

type qrGenerateRequest struct {
	Type string `json:"type"`

	URL   string `json:"url"`
	Text  string `json:"text"`
	Phone string `json:"phone"`

	WiFi  qrWiFiRequest  `json:"wifi"`
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
