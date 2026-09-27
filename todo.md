# Media Converter – TODO

## 1. Medien-Konvertierung

### Dokumente

- [x] `.rtf` als Dokument erkennen und unterstützen
  - [x] MIME-/Dateityp-Erkennung ergänzen
  - [x] Konvertierung über LibreOffice
  - [x] sinnvolle Zielformate definieren
  - [x] PDF-Ausgabe testen
  - [x] DOCX-/ODT-Ausgabe testen

- [x] `.txt` als Dokument erkennen und unterstützen
  - [x] MIME-/Dateityp-Erkennung ergänzen
  - [ ] Zeichencodierung explizit validieren / absichern
  - [x] TXT → PDF
  - [x] TXT → DOCX
  - [x] TXT → ODT
  - [ ] optional TXT → HTML

### Markdown

- [x] Markdown als Eingabeformat der Medien-Konvertierung unterstützen
  - [x] `.md`, `.mdx` und `.markdown` erkennen
  - [x] sichere Markdown-Verarbeitung über Goldmark
  - [x] Raw HTML nicht ausführen
  - [x] externe Ressourcen kontrollieren / Markdown-Bilder blockieren

- [x] Markdown → PDF
  - [x] Markdown → HTML
  - [x] HTML über Chromium als PDF rendern
  - [x] Tabellen
  - [x] Task Lists
  - [x] Codeblöcke
  - [x] Blockquotes
  - [x] Links
  - [x] Seitenumbrüche / Druck-CSS

- [x] Markdown → Bild
  - [x] Markdown über Chromium rendern
  - [x] PNG
  - [x] JPEG
  - [x] WEBP
  - [x] automatische Höhe anhand des Inhalts
  - [x] maximale Renderhöhe definieren
  - [x] sehr lange Dokumente kontrolliert ablehnen

- [x] Markdown → Website
  - [x] vollständige HTML-Datei erzeugen
  - [x] eingebettetes CSS
  - [x] keine extern erforderlichen Assets
  - [x] Dokumenttitel aus erster H1 übernehmen

### Medien-Konverter UX

- [ ] Unterstützte Eingabeformate vollständig in der UI anzeigen
- [x] Unterstützte Zielformate abhängig vom erkannten Eingabeformat anzeigen
- [x] Batch-Konvertierung mehrerer Dateien gleichen Typs
  - [x] maximal 20 Dateien gleichzeitig
  - [x] gemischte Eingabeformate ablehnen
  - [x] mehrere Ergebnisse automatisch als ZIP ausgeben
  - [x] Drag-and-Drop mehrerer Medien
- [ ] verständlichere Fehlermeldung bei nicht unterstützten Dateitypen weiter vereinheitlichen
- [ ] Dateiendung und tatsächlichen MIME-/Dateityp gegeneinander validieren
- [x] Ausgabe-Dateinamen konsistent aus dem ursprünglichen Dateinamen ableiten


## 2. PDF-Werkzeuge

### Noch offene Werkzeuge

- [ ] PDF bearbeiten
  - [ ] Funktionsumfang definieren
  - [ ] Text hinzufügen
  - [ ] Textfelder positionieren
  - [ ] Bilder hinzufügen
  - [ ] ggf. Freihand / Zeichnen
  - [ ] bestehende PDF-Seite als Hintergrund verwenden
  - [ ] Änderungen verlustfrei auf PDF anwenden

- [ ] PDF schwärzen
  - [ ] PDF hochladen
  - [ ] Seitenvorschau anzeigen
  - [ ] Bereiche per Maus markieren
  - [ ] mehrere Schwärzungen pro Seite
  - [ ] Markierungen entfernen / ändern
  - [ ] echte Redaction statt nur schwarzer Overlay-Fläche
  - [ ] darunterliegenden Text/Inhalt dauerhaft entfernen
  - [ ] Metadaten / versteckte Inhalte berücksichtigen
  - [ ] Ergebnis kontrollieren

### Vorhandene PDF-Werkzeuge – Nacharbeiten

- [ ] PDF-Seiten sortieren
  - [ ] Drag-and-Drop insbesondere bei benachbarten Seiten nochmals testen

- [ ] PDF trennen
  - [ ] Performance bei sehr vielen erzeugten Einzeldateien verbessern
  - [ ] nicht für jeden Teil unnötig einen separaten qpdf-Prozess starten, falls vermeidbar

- [ ] PDF komprimieren
  - [ ] parallele Analyse desselben Uploads/Modus gegen doppelte Berechnung absichern
  - [ ] Ghostscript-Ergebnisse mit problematischen PDFs weiter testen
  - [ ] Formulare, Transparenzen und eingebettete Fonts testen

- [ ] PDF optimieren
  - [ ] Fast-Web-View / Linearization mit größeren PDFs testen
  - [ ] bereits optimierte PDFs testen
  - [ ] PDFs mit Formularen und Signaturen testen

- [ ] PDF verschlüsseln / entschlüsseln
  - [ ] Passwortübergabe an qpdf prüfen
  - [ ] Passwörter möglichst nicht über Prozess-Argumente weitergeben
  - [ ] qpdf Password-File-Unterstützung verwenden, sofern geeignet
  - [ ] Passwortlängen entsprechend qpdf/PDF-Spezifikation validieren
  - [ ] verschiedene AES-256-PDFs testen

- [ ] Webseite → PDF
  - [x] Desktop-Modus testen
  - [x] Tablet-Modus testen
  - [x] Mobil-Modus testen
  - [x] Druckansicht testen
  - [x] responsive Breakpoints mit realen Webseiten testen
  - [ ] Weiterleitungen sicher behandeln
  - [ ] SSRF-Schutz auch nach Redirects sicherstellen
  - [ ] DNS-Rebinding berücksichtigen
  - [ ] Ziel-IP unmittelbar vor Verbindung erneut prüfen
  - [ ] lokale Server-IP weiterhin blockieren
  - [ ] localhost / Loopback weiterhin blockieren

- [ ] PDF erstellen
  - [ ] Markdown-Modus mit größeren Dokumenten testen
  - [ ] Tabellen über Seitenumbrüche testen
  - [ ] lange Codeblöcke testen
  - [ ] sehr lange URLs / Wörter testen
  - [x] Markdown-Bilder weiterhin blockieren
  - [x] Raw HTML weiterhin nicht ausführen


## 3. Sicherheit

### Web-/Chromium-Rendering

- [ ] SSRF-Schutz zentralisieren
- [ ] localhost blockieren
- [ ] IPv4-Loopback `127.0.0.0/8` blockieren
- [ ] IPv6-Loopback `::1` blockieren
- [ ] alle IP-Adressen des Media-Converter-Servers blockieren
- [ ] DNS-Namen blockieren, die auf den Media-Converter-Server zeigen
- [ ] Redirect-Ziele ebenfalls validieren
- [ ] DNS-Rebinding absichern
- [ ] Chromium-Netzwerkzugriffe auf Subresources bewerten
  - [ ] Bilder
  - [ ] CSS
  - [ ] Fonts
  - [ ] JavaScript
  - [ ] iframes
- [ ] entscheiden, ob weitere sensible Netze explizit blockiert werden sollen
  - [ ] Link-local
  - [ ] Cloud-Metadata-Adressen
  - [ ] Docker-/Container-Netze
  - [ ] Proxmox-/Management-Netze

### Uploads

- [ ] Upload-Dateitypen nicht ausschließlich anhand der Dateiendung akzeptieren
- [ ] Magic Bytes / Signaturen vollständig prüfen
- [ ] beschädigte Dateien sauber ablehnen
- [ ] Zip-Bomb-/Decompression-Bomb-Risiken prüfen
- [ ] maximale entpackte/verarbeitete Dateigrößen definieren


## 4. Upload- und Dateiverwaltung

- [ ] Upload-Store gegen Race Conditions absichern
  - [ ] laufende Verarbeitung darf nicht vom Cleanup gelöscht werden
  - [ ] Acquire/Release bzw. Lease-System einführen
  - [ ] Cleanup nur unbenutzte Uploads entfernen

- [ ] PDF-Upload-Store ebenfalls mit Lease-System versehen

- [ ] temporäre Dateien nach Erfolg zuverlässig löschen
- [ ] temporäre Dateien nach Fehler zuverlässig löschen
- [ ] temporäre Dateien nach Client-Abbruch zuverlässig löschen
- [ ] Cleanup beim Server-Shutdown testen

- [ ] einheitliche Lebensdauer für temporäre Uploads dokumentieren


## 5. Downloads und große Dateien

- [ ] große Downloads nicht vollständig mit `fetch().blob()` im Browser puffern
- [ ] Download-Token bzw. temporären Download-Endpunkt implementieren
- [ ] Browser direkt auf Download-Endpunkt navigieren lassen
- [ ] große Dateien streamen
- [ ] Content-Length soweit möglich setzen
- [ ] Client-Abbruch beim Streaming korrekt behandeln

- [ ] bestehende PDF-Handler vereinheitlichen
  - [ ] Quelldatei erst nach erfolgreichem Ergebnis löschen
  - [ ] Fehler während des Downloads korrekt behandeln


## 6. Parallelisierung und Ressourcen

- [ ] globale Conversion-Slots überprüfen
- [ ] unterschiedliche Workloads ggf. getrennt limitieren
  - [ ] FFmpeg
  - [ ] ImageMagick
  - [ ] Ghostscript
  - [ ] Chromium
  - [ ] LibreOffice
  - [ ] qpdf

- [ ] Chromium-Prozesse aufräumen, wenn Request abgebrochen wird
- [ ] Zombie-Prozesse verhindern
- [ ] Speicherverbrauch bei großen Dateien beobachten
- [ ] maximale CPU-/RAM-Auslastung auf Produktionsserver testen

- [ ] Preview-Rendering prüfen
  - [ ] PDF-Previews konkurrieren aktuell mit normalen Conversion-Slots
  - [ ] ggf. eigenen Preview-Pool verwenden


## 7. Fehlerbehandlung

- [ ] Fehlerantworten vereinheitlichen
- [ ] strukturierte Fehlercodes für Frontend einführen
- [x] technische Details nur serverseitig loggen
- [x] Benutzer bekommt kurze verständliche Fehlermeldung bei Medien-Konvertierungen
- [ ] Fehlertexte aller PDF-Werkzeuge vereinheitlichen
- [ ] Timeout-Fehler einheitlich behandeln
- [ ] Queue-Timeout einheitlich behandeln
- [x] Client-Abbruch im zentralen Request-Logging nicht als Serverfehler behandeln

- [ ] Fehler externer Programme vereinheitlichen
  - [ ] FFmpeg
  - [ ] ImageMagick
  - [ ] LibreOffice
  - [ ] Ghostscript
  - [ ] qpdf
  - [ ] Poppler
  - [ ] Tesseract
  - [ ] Chromium


## 8. Logging und Observability

- [x] zentrale HTTP-Request-Logs
  - [x] Method
  - [x] Path
  - [x] Status
  - [x] Dauer
  - [x] WARN/ERROR mit erweitertem Diagnosekontext

- [x] INFO-Logs kompakt einzeilig darstellen
  - [x] technische Felder bei Aktionslogs ausblenden
  - [x] maximal sechs Felder pro INFO-Aktionslog
  - [x] lange Feldnamen für INFO kürzen
  - [x] WARN und ERROR weiterhin ausführlich darstellen

- [x] Conversion-/Aktionslogs vereinheitlichen
  - [x] Medienerkennung loggen
  - [x] Medien-Konvertierungen loggen
  - [x] QR-Erstellung loggen
  - [x] vorhandene PDF-Aktionslogs zentral kompakt darstellen

- [ ] Logging auf vertrauliche Daten vollständig auditieren
  - [ ] keine vertraulichen Inhalte loggen
  - [ ] keine Passwörter loggen
  - [ ] keine vollständigen URLs mit Tokens/Query-Strings loggen
  - [x] Dateiinhalt niemals loggen

- [ ] sinnvolle Werte in allen Werkzeugen vollständig vereinheitlichen
  - [x] Dateityp / Quellformat
  - [ ] Eingabegröße überall
  - [ ] Ausgabegröße überall
  - [x] Conversion-Typ / Zielformat
  - [x] Erfolg / Fehler
  - [x] Timeout

- [ ] optional einfache interne Statusseite
  - [ ] aktive Conversions
  - [ ] belegte Worker
  - [ ] temporärer Speicherverbrauch
  - [ ] verfügbare Converter


## 9. Abhängigkeiten und Systemprüfung

- [ ] beim Start alle erforderlichen Tools prüfen
  - [ ] FFmpeg
  - [ ] ffprobe
  - [ ] ImageMagick
  - [ ] LibreOffice
  - [ ] Poppler
  - [ ] Ghostscript
  - [ ] qpdf
  - [ ] Tesseract
  - [ ] Chromium / Chrome / Edge

- [ ] Versionen der erkannten Tools beim Start loggen
- [ ] verständliche Fehlermeldung bei fehlender Dependency
- [ ] optionale vs. zwingende Dependencies unterscheiden

- [ ] Debian-Installationsanleitung aktualisieren
- [ ] Ghostscript explizit in Debian-Abhängigkeiten aufnehmen
- [ ] Chromium explizit dokumentieren


## 10. Tests

### Backend

- [ ] Tests für Medienerkennung erweitern
- [ ] `.txt`
- [ ] `.rtf`
- [ ] `.md`
- [ ] `.mdx`
- [ ] falsche Dateiendungen
- [ ] beschädigte Dateien

- [ ] Tests für Markdown
  - [ ] Überschriften
  - [ ] Tabellen
  - [ ] Task Lists
  - [ ] Codeblöcke
  - [ ] Raw HTML
  - [ ] verbotene Bilder
  - [ ] gefährliche Links

- [ ] Tests für Batch-Konvertierung
  - [ ] mehrere Dateien gleichen Typs
  - [ ] gemischte Formate ablehnen
  - [ ] maximal 20 Dateien
  - [ ] doppelte Dateinamen im ZIP
  - [ ] Einzeldatei weiterhin ohne ZIP
  - [ ] Abbruch bei Fehler einer Batch-Datei

- [ ] Tests für Web-PDF-Sicherheit
  - [ ] localhost
  - [ ] `127.0.0.1`
  - [ ] anderes `127.x.x.x`
  - [ ] `::1`
  - [ ] eigene Server-IP
  - [ ] DNS → eigene Server-IP
  - [ ] Redirect → localhost
  - [ ] Redirect → eigene Server-IP
  - [ ] DNS-Rebinding

- [ ] Tests für PDF Encrypt/Decrypt
- [ ] Tests für PDF Compression
- [ ] Tests für PDF Optimize
- [ ] Tests für PDF Split/Merge/Sort/Rotate/Delete/Extract

### Frontend

- [ ] TypeScript-Tests für Format-/Input-Validierung
- [x] Multi-File-Drag-and-Drop testen
- [ ] Upload-Abbruch testen
- [x] Wechsel zwischen PDF-Werkzeugen testen
- [ ] mehrfaches Öffnen/Schließen der Workspaces testen
- [ ] parallele Requests testen


## 11. UI / UX

- [ ] einheitliche Drop-Zones
- [ ] einheitliche Progress-Anzeigen
- [ ] einheitliche Fehlerboxen
- [ ] einheitliche Ergebnis-/Downloadbereiche

- [ ] gemeinsame Frontend-Komponenten stärker wiederverwenden
  - [ ] Upload
  - [ ] Drop-Zone
  - [ ] Progress
  - [ ] Error
  - [ ] Download
  - [ ] Seitenauswahl

- [ ] Accessibility prüfen
  - [ ] Tastaturbedienung
  - [ ] Focus States
  - [ ] Labels
  - [ ] ARIA bei dynamischen Statusmeldungen
  - [ ] ausreichender Kontrast

- [ ] Mobile Layout aller Werkzeuge prüfen


## 12. Code-Qualität

- [x] 500-LOC-Limit automatisiert über `cmd/checkloc` / `just check` durchsetzen
- [x] `convert.go` in kleinere Verantwortlichkeiten aufteilen
- [ ] weitere große Dateien frühzeitig aufteilen
- [ ] doppelte PDF-Frontend-Logik reduzieren
- [ ] doppelte Selection-Logik reduzieren
- [ ] gemeinsame Request-/Response-Helfer prüfen
- [ ] gemeinsame Streaming-Helfer für Downloads erstellen
- [ ] gemeinsame Temp-Directory-Helfer prüfen
- [ ] gemeinsame Timeout-/Conversion-Slot-Behandlung prüfen
- [ ] Markdown-Rendering zwischen Medien-Konverter und PDF-Erstellung weiter deduplizieren

- [ ] ungenutzten Code regelmäßig mit `gopls` / Compiler bereinigen
- [ ] `go vet` in `just check` prüfen/integrieren
- [ ] `staticcheck` optional integrieren


## 13. Deployment / Produktion

- [ ] Produktionsinstallation auf Debian vollständig dokumentieren
- [ ] benötigte Debian-Pakete dokumentieren
- [ ] Chromium-Setup dokumentieren
- [ ] `CHROME_BIN` dokumentieren
- [ ] Dateisystemrechte für Temp-Verzeichnisse prüfen

- [ ] systemd-Service erstellen/dokumentieren
- [ ] eigener unprivilegierter Benutzer für Media Converter
- [ ] Restart-Policy
- [ ] Resource Limits
- [ ] Temp-Verzeichnis / Speicherlimits
- [ ] Log-Rotation

- [ ] Reverse-Proxy-Konfiguration dokumentieren (nginx proxy manager)
- [ ] maximale Upload-Größe im Reverse Proxy passend konfigurieren
- [ ] Request-/Proxy-Timeouts für lange Conversions abstimmen

- [ ] Produktions-Test mit großen Dateien
  - [ ] 100 MiB
  - [ ] 250 MiB
  - [ ] 500 MiB


## 14. Später / optionale Erweiterungen

- [x] Batch-Konvertierung mehrerer Dateien gleichen Typs
- [x] mehrere Ergebnisse als ZIP herunterladen
- [x] Drag-and-Drop mehrerer Medien
- [ ] Conversion-Presets
- [ ] zuletzt verwendete Optionen lokal im Browser merken
- [ ] Bild-Metadaten entfernen
- [ ] Audio-Metadaten entfernen
- [ ] Video-Metadaten entfernen
- [ ] PDF-Metadaten anzeigen / entfernen
- [ ] OCR für gescannte PDFs als eigenes Werkzeug
- [ ] Bilder → durchsuchbare OCR-PDF
- [ ] PDF → Bilder als eigenes Werkzeug
- [ ] Bilder → PDF als eigenes Werkzeug
