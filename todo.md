# Media Converter – TODO

## 1. Medien-Konvertierung

### Dokumente

- [ ] `.rtf` als Dokument erkennen und unterstützen
  - [ ] MIME-/Dateityp-Erkennung ergänzen
  - [ ] Konvertierung über LibreOffice
  - [ ] sinnvolle Zielformate definieren
  - [ ] PDF-Ausgabe testen
  - [ ] DOCX-/ODT-Ausgabe testen

- [ ] `.txt` als Dokument erkennen und unterstützen
  - [ ] MIME-/Dateityp-Erkennung ergänzen
  - [ ] Zeichencodierung sauber behandeln
  - [ ] TXT → PDF
  - [ ] TXT → DOCX
  - [ ] TXT → ODT
  - [ ] optional TXT → HTML

### Markdown

- [ ] Markdown als Eingabeformat der Medien-Konvertierung unterstützen
  - [ ] `.md`, `.mdx` und `.markdown` erkennen
  - [ ] sichere Markdown-Verarbeitung über Goldmark
  - [ ] Raw HTML nicht ausführen
  - [ ] externe Ressourcen kontrollieren

- [ ] Markdown → PDF
  - [ ] Markdown → HTML
  - [ ] HTML über Chromium als PDF rendern
  - [ ] Tabellen
  - [ ] Task Lists
  - [ ] Codeblöcke
  - [ ] Blockquotes
  - [ ] Links
  - [ ] Seitenumbrüche / Druck-CSS

- [ ] Markdown → Bild
  - [ ] Markdown über Chromium rendern
  - [ ] PNG
  - [ ] JPEG
  - [ ] WEBP
  - [ ] automatische Höhe anhand des Inhalts
  - [ ] maximale Renderhöhe definieren
  - [ ] sehr lange Dokumente sinnvoll behandeln

- [ ] Markdown → Website
  - [ ] vollständige HTML-Datei erzeugen
  - [ ] eingebettetes CSS
  - [ ] keine extern erforderlichen Assets
  - [ ] optional Dokumenttitel aus erster H1 übernehmen

### Medien-Konverter UX

- [ ] Unterstützte Eingabeformate vollständig in der UI anzeigen
- [ ] Unterstützte Zielformate abhängig vom erkannten Eingabeformat anzeigen
- [ ] verständlichere Fehlermeldung bei nicht unterstützten Dateitypen
- [ ] Dateiendung und tatsächlichen MIME-/Dateityp gegeneinander validieren
- [ ] Ausgabe-Dateinamen konsistent aus dem ursprünglichen Dateinamen ableiten


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
  - [ ] Markdown-Bilder weiterhin blockieren
  - [ ] Raw HTML weiterhin nicht ausführen


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
- [ ] technische Details nur serverseitig loggen
- [ ] Benutzer bekommt kurze verständliche Fehlermeldung
- [ ] Timeout-Fehler einheitlich behandeln
- [ ] Queue-Timeout einheitlich behandeln
- [ ] Client-Abbruch nicht als Serverfehler loggen

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

- [ ] Conversion-Logs vereinheitlichen
- [ ] keine vertraulichen Inhalte loggen
- [ ] keine Passwörter loggen
- [ ] keine vollständigen URLs mit Tokens/Query-Strings loggen
- [ ] Dateiinhalt niemals loggen

- [ ] sinnvolle Werte loggen
  - [ ] Dateityp
  - [ ] Eingabegröße
  - [ ] Ausgabegröße
  - [ ] Conversion-Typ
  - [ ] Erfolg / Fehler
  - [ ] Timeout

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
- [ ] Upload-Abbruch testen
- [ ] Wechsel zwischen PDF-Werkzeugen testen
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

- [ ] 500-LOC-Limit weiterhin einhalten
- [ ] große Dateien frühzeitig aufteilen
- [ ] doppelte PDF-Frontend-Logik reduzieren
- [ ] doppelte Selection-Logik reduzieren
- [ ] gemeinsame Request-/Response-Helfer prüfen
- [ ] gemeinsame Streaming-Helfer für Downloads erstellen
- [ ] gemeinsame Temp-Directory-Helfer prüfen
- [ ] gemeinsame Timeout-/Conversion-Slot-Behandlung prüfen

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

- [ ] Batch-Konvertierung mehrerer Dateien
- [ ] mehrere Ergebnisse als ZIP herunterladen
- [ ] Drag-and-Drop mehrerer Medien
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
