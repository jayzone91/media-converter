# Media Converter

## Benötigte Software

Folgende Programme müssen installiert und über `PATH` erreichbar sein:

- Git
- Go
- Bun
- just
- templ CLI
- FFmpeg
- ffprobe
- ImageMagick
- LibreOffice
- Poppler
- Tesseract OCR
- qpdf
- Ghostscript

Abhängigkeiten installieren:

```powershell
bun install
go install github.com/a-h/templ/cmd/templ@v0.3.1020
go mod download
```

## Temporäre Uploads

Temporäre Medien- und PDF-Uploads werden grundsätzlich für maximal 30 Minuten vorgehalten.

Ein Hintergrund-Cleanup entfernt abgelaufene Uploads regelmäßig. PDF-Uploads, die sich aktuell in Verarbeitung befinden, sind während dieser Verarbeitung vor dem Cleanup geschützt.

Beim Herunterfahren des Servers werden noch vorhandene temporäre Uploads entfernt.
