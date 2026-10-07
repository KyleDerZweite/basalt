# Idee: Personenprofile auf einer Karte

Stand: 7. Oktober 2026. Status: unverbindliche Idee, vorerst zurückgestellt. Es gibt keinen Implementierungsplan, keine gewählte Architektur und keinen Auftrag für einen Prototyp.

Die Idee verbindet Basalts Personenprofile und Accounts mit belegten Ortsangaben auf einer Karte. Als mögliche Integrationen wurden WorldWideView und God's Eye View untersucht. Leonardos God-s-eye wurde als mögliche Quelle für zusätzliche Plattformprüfungen betrachtet.

## Warum vorerst zurückgestellt

Der zusätzliche Erkenntniswert eines Globe ist bisher unklar. Basalts Nutzen liegt vor allem in Accounts, Identitätsbelegen und Verbindungen. Allgemeine Profilangaben wie "Berlin" oder "Germany" rechtfertigen allein keine separate Kartenintegration. Geocoding, Ortsgenauigkeit und die Pflege eines zweiten Projekts würden zusätzlichen Aufwand verursachen.

Die Idee kann erneut betrachtet werden, wenn ein konkreter Recherchefall zeigt, welche räumliche Frage ein Dossier oder Graph nicht ausreichend beantwortet. Eine Ortsliste oder kleine Karte innerhalb eines Dossiers wäre dann ebenfalls eine Option. Auch diese Alternative ist nicht zur Umsetzung ausgewählt.

## Technische Optionen aus der Recherche

Die folgenden Befunde halten mögliche Ansätze fest. Sie legen weder eine Reihenfolge noch einen nächsten Arbeitsschritt fest. Die Anwendungen wurden nicht gestartet, und ihre Tests wurden nicht ausgeführt.

Falls ein eigenständiges Basalt Plugin später sinnvoll wird, wäre WorldWideView technisch der direkteste Kandidat. Das Projekt besitzt einen konkreten Plugin Lifecycle, ein geographisches Datenmodell und Erweiterungspunkte für Detailansichten. Basalt sollte seine Scans und Daten weiterhin selbst verwalten. Ein Adapter übersetzt ausgewählte Ergebnisse für den Viewer.

Für eine eigene, in Basalt gestaltete Kartenansicht ist Bilawals God’s Eye View eine zweite Option. Das aktuelle Projekt exportiert wiederverwendbare Application und Layer Komponenten. Es besitzt aber keine automatische Plugin Discovery. Die Wahl hängt deshalb davon ab, ob Basalt in einem vorhandenen Viewer erscheinen oder einen eigenen Viewer zusammensetzen soll.

Leonardos God’s Eye liefert hauptsächlich Anregungen für Username Enumeration. Sein aktueller Code ergänzt weder ein Kartenmodell noch eine belastbare Zuordnung von Accounts zu Personen.

## Beobachteter Stand

Die Recherche verwendet folgende Revisionen der jeweiligen Default Branches:

- Basalt: `aeef1362bba5f64febcd5ad09fb1f9625921e99a`, lokaler Branch `t3/integrate-basalt-worldwideview`. Der Arbeitsbaum war vor dieser Notiz sauber.
- [WorldWideView](https://github.com/silvertakana/worldwideview/commit/33df7a5b7f89c9d9d432400937fa45aa7481d683): `33df7a5b7f89c9d9d432400937fa45aa7481d683`, Commitdatum 26. September 2026.
- [God’s Eye View von Bilawal](https://github.com/bilawalsidhu/gods-eye-view/commit/7cbe04f515282019a53d03674867d286152f973e): `7cbe04f515282019a53d03674867d286152f973e`, Commitdatum 7. Oktober 2026.
- [God’s Eye von Leonardo](https://github.com/LeonardoCides/God-s-eye/commit/6b251a561f311169c45477345f606a11555dfeda): `6b251a561f311169c45477345f606a11555dfeda`, Commitdatum 15. Februar 2026.

## WorldWideView

Das Projekt verwendet Next.js, React, TypeScript und CesiumJS. Das SDK definiert `WorldPlugin` mit `initialize`, `destroy`, `fetch`, Polling und Rendering. `GeoEntity` verlangt ID, Plugin ID, WGS84 Koordinaten, Timestamp und Properties. Optionale React Komponenten erweitern Sidebar, Details, Settings, Globe und Bottom Panel. Diese Erweiterungspunkte sind im [SDK Quellcode](https://github.com/silvertakana/worldwideview/blob/33df7a5b7f89c9d9d432400937fa45aa7481d683/packages/wwv-plugin-sdk/src/index.ts) vorhanden.

Der [PluginManager](https://github.com/silvertakana/worldwideview/blob/33df7a5b7f89c9d9d432400937fa45aa7481d683/src/core/plugins/PluginManager.ts) lädt Plugins aus Manifests, registriert sie, ruft `initialize` auf und verbindet Updates mit Cache und DataBus. Die Plugin Unterstützung ist damit mehr als eine README Ankündigung. Der [Quickstart](https://github.com/silvertakana/worldwideview/blob/33df7a5b7f89c9d9d432400937fa45aa7481d683/docs/plugin-quickstart.md) beschreibt lokale Plugin Entwicklung. Der [Advanced Guide](https://github.com/silvertakana/worldwideview/blob/33df7a5b7f89c9d9d432400937fa45aa7481d683/docs/plugin-advanced.md) beschreibt dynamische ES Module Bundles und eigene Backends. Diese Abläufe wurden nicht praktisch verifiziert.

Die [LICENSE](https://github.com/silvertakana/worldwideview/blob/33df7a5b7f89c9d9d432400937fa45aa7481d683/LICENSE) enthält Elastic License 2.0. Sie schränkt insbesondere das Anbieten wesentlicher Softwarefunktionen als Hosted oder Managed Service ein. Das ist ein konkreter Unterschied zu einer MIT Basis für ein späteres Hostingprodukt. Diese Recherche bewertet keine abschließende Lizenzkompatibilität mit Basalt.

## God’s Eye View von Bilawal

Das [package.json](https://github.com/bilawalsidhu/gods-eye-view/blob/7cbe04f515282019a53d03674867d286152f973e/package.json) beschreibt eine JavaScript ESM Anwendung mit Vite und Cesium. Es exportiert Application, Viewer, einzelne Layer und weitere Komponenten. Der [Application Controller](https://github.com/bilawalsidhu/gods-eye-view/blob/7cbe04f515282019a53d03674867d286152f973e/src/app/application.js) nimmt Konstruktoren für Scene, Controls, Data und Tools entgegen und verwaltet Start, Abbruch und Cleanup. Ein [Layer Adapter](https://github.com/bilawalsidhu/gods-eye-view/blob/7cbe04f515282019a53d03674867d286152f973e/src/app/layers/earthquakes.js) zeigt die explizite Verbindung zwischen Layer und Overlay Host.

Laut [Application Dokumentation](https://github.com/bilawalsidhu/gods-eye-view/blob/7cbe04f515282019a53d03674867d286152f973e/docs/APPLICATION.md) gibt es keine Module Discovery oder automatische Imports. Die bestehende Standalone Shell besitzt seitengebundenen Zustand und ist keine frei einsetzbare, entfernbare HTML Shell für mehrere Viewer. Daraus folgt als technische Einschätzung: Ein Basalt Adapter lässt sich explizit komponieren, benötigt aber eigene Integration und UI Ownership. Ein kompletter Fork ist dadurch nicht automatisch nötig.

Der Quellcode steht laut [LICENSE](https://github.com/bilawalsidhu/gods-eye-view/blob/7cbe04f515282019a53d03674867d286152f973e/LICENSE) unter MIT. Dieselbe Datei grenzt Drittanbieterdaten und Modelle ausdrücklich aus. Einzelne mitgelieferte Datensätze haben beispielsweise NonCommercial Bedingungen. Das Projekt enthält Tests und Boundary Checks im Package, aber deren Vorhandensein belegt keinen erfolgreichen Lauf oder stabile externe API.

## God’s Eye von Leonardo

Das Repository enthält an der geprüften Revision `README.md`, `god.png` und `script.py`. Der [Python Code](https://github.com/LeonardoCides/God-s-eye/blob/6b251a561f311169c45477345f606a11555dfeda/script.py) prüft 13 Profil URL Muster parallel. Er folgt Redirects und behandelt HTTP 200 als Treffer. Netzwerkfehler verschwinden ohne eigenes Ergebnis. Die Ausgabe besteht aus gefundenen URLs und einer Textdatei.

Ein HTTP 200 belegt für sich weder einen existierenden Account noch dessen Zugehörigkeit zur gesuchten Person. Der Code enthält keine Koordinaten, Geocodierung, Personenauflösung oder Plugin Schnittstelle. Eine Übernahme sollte deshalb höchstens nachweislich fehlende Plattformprüfungen inspirieren. Basalts vorhandene Module sollten die Prüfungen mit plattformspezifischen Positivsignalen und Fehlerzuständen implementieren.

Das [README](https://github.com/LeonardoCides/God-s-eye/blob/6b251a561f311169c45477345f606a11555dfeda/README.md) zeigt ein MIT Badge. Eine LICENSE Datei fehlt in der geprüften Revision. Das Badge allein ist keine ausreichende Grundlage, um den Code als eindeutig MIT lizenziert einzuordnen.

## Mögliche Integrationsgrenze

Basalt besitzt bereits eine [lokale API](local-api.md) mit Targets, Scans, Ergebnissen und Event Streams. Bearer Auth und erlaubte Browser Origins sind vorgesehen. Der [Target Typ](../cli/internal/app/types.go) kann ein Dossier verankern. Die [Graph Node Typen](../cli/internal/graph/node.go) besitzen derzeit keinen expliziten Personen oder Location Typ. `Properties` sind frei, und der [Workspace](../cli/internal/app/workspace.go) verwendet `location` als Text.

Bei einer späteren Umsetzung wäre ein eigener Datenvertrag für Ortsbelege nötig. Eine Kartenbeobachtung sollte mindestens Target ID, verknüpften Account oder Node, Quelle, Koordinaten, Ortsgenauigkeit, Confidence, Beobachtungszeit und Art des Ortsbezugs enthalten. Account Zuordnungen zu einer Person brauchen einen ausdrücklichen Bestätigungsstatus. Eine Profilangabe wie Berlin, ein beobachteter Aufenthaltsort und die Position eines Servers brauchen unterschiedliche Bedeutungen. Die Geocodierung eines Ortsnamens erzeugt keinen Beleg für einen aktuellen Personenstandort. Eine Infrastruktur IP darf keine Personenkoordinaten liefern. Targets ohne belastbaren Ortsbezug bleiben im Dossier sichtbar, ohne einen erfundenen Kartenpunkt zu erhalten.

Eine mögliche Variante wäre ein lesender WorldWideView Adapter. Er würde ausgewählte Targets und Ortsbelege über Basalts API laden, sie in `GeoEntity` übersetzen und bei Auswahl Quelle, Zeitbezug und Accounts anzeigen. Basalt bleibt Eigentümer von Scan Lifecycle, Speicherung und Bewertung. Der Kartenadapter ist ein API Consumer und implementiert kein `Module.Extract`.

Ein lokaler Betrieb beider Anwendungen ist für einen ersten Versuch naheliegend. Bei einer entfernten Viewer Instanz muss der Datenweg zum lokalen Basalt ausdrücklich entworfen werden. Browser Erreichbarkeit, Origin Konfiguration und Token Handhabung gehören zu diesem Vertrag. REST Polling reicht für einen ersten lesenden Versuch. Bestehende SSE Events können später Aktualisierungen auslösen, ohne ungeprüft einen WebSocket Backend Dienst hinzuzufügen.

Der festgehaltene Anwendungsfall ist ein Personenprofil mit Accounts und belegten Ortsangaben. Ob dafür ein externer Viewer oder eine Ansicht innerhalb von Basalt sinnvoll ist, bleibt offen. Ein Prototyp ist nicht geplant. Zeitlich verfolgte Beobachtungen wären ein zusätzlicher Anwendungsfall mit eigenen Quellen und einem Zeitvertrag. Keine der untersuchten Integrationen erzeugt diese Evidenz automatisch.
