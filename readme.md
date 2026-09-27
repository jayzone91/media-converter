## Benötigte Software

just
templ cli
ffmpeg + ffprobe
imagemagick
libreoffice
poppler
tesseract ocr
qpdf
ghostscript


bun install

go install github.com/a-h/templ/cmd/templ@v0.3.1020

go mod download



git --version
go version
bun --version
just --version
templ version

ffmpeg -version
ffprobe -version
magick -version
soffice --version
pdftotext -v
pdftoppm -v
tesseract --version
qpdf --version
gswin64c --version


Achte darauf, dass die Programme im PATH liegen. Besonders häufig problematisch:
- soffice.exe
- pdftotext.exe
- pdftoppm.exe
- gswin64c.exe
- qpdf.exe
- tesseract.exe
