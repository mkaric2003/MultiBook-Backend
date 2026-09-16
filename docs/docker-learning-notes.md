# Docker bilješke

Kratke bilješke o Dockeru na MultiBook projektu.

## Osnovni pojmovi

Docker pokreće procese u izolovanim, ponovljivim okruženjima.

```text
Dockerfile → recept za image
Image      → nepromjenjivi paket aplikacije i njenog runtimea
Container  → pokrenuta instanca imagea
Volume     → trajni podaci izvan životnog vijeka containera
Port map   → povezuje port host računara s portom containera
Compose    → opisuje više povezanih servisa
```

MultiBook lokalno koristi:

```text
postgis/postgis:17-3.5 → PostgreSQL + PostGIS image
postgres container      → pokrenuta lokalna baza
postgres-data volume    → trajno čuva DB fajlove
5432:5432               → localhost:5432 prosljeđuje u container:5432
migrate container       → jednokratno izvršava SQL migracije
```

Brisanje ili ponovno kreiranje containera ne briše named volume. Brisanje volumea briše lokalne podatke baze.

## Docker portovi

Svaki container ima vlastitu mrežu i svoj `localhost`. Port mapping koristi format:

```text
HOST_PORT:CONTAINER_PORT
```

```yaml
ports:
  - "5432:5432"
```

znači:

```text
Mac localhost:5432 → PostgreSQL container:5432
```

Portovi ne moraju imati isti broj:

```text
15432:5432 → Mac koristi 15432, PostgreSQL u containeru i dalje koristi 5432
```

Komunikacija zavisi od mjesta klijenta:

```text
Go pokrenut na Macu         → postgres://...@localhost:5432/...
Go u istom Compose networku → postgres://...@postgres:5432/...
```

Unutar API containera `localhost` označava taj API container, a ne Mac niti PostgreSQL container.

`EXPOSE 8080` u Dockerfileu dokumentuje da aplikacija sluša na portu 8080; sam ne objavljuje port host računaru. Za pristup s hosta i dalje je potreban `ports`, npr. `8080:8080`, ili `docker run -p 8080:8080`.

## Docker Compose

Compose fajl deklarativno opisuje lokalni sistem: servise, njihove imagee, konfiguraciju, mrežu, zavisnosti i volumene. **Service** je definicija, a **container** je pokrenuta instanca te definicije.

```text
docker-compose.yml
    ├── postgres service → dugotrajni PostgreSQL container
    └── migrate service  → jednokratni migration container
```

`docker compose up -d postgres`:

```text
pročitaj Compose → kreiraj mrežu/volume ako nedostaju → kreiraj ili uskladi container → pokreni ga u pozadini
```

- `environment` – varijable unutar containera.
- `healthcheck` – provjera je li servis stvarno spreman, ne samo pokrenut.
- `depends_on` – redoslijed/zavisnost između servisa.
- `profiles` – opcionalni servisi koji se ne pokreću običnim `up` pozivom.
- Compose automatski pravi zajedničku mrežu; naziv servisa postaje hostname, npr. `postgres:5432`.

Compose servis bez `profiles` je uvijek omogućen. Servis s `profiles: [tools]` preskače se u običnom `docker compose up`, a uključuje se s `--profile tools` ili kada se taj servis eksplicitno navede kao cilj. Profile je oznaka za grupisanje opcionalnih servisa, ne poseban container, image ili environment.

```text
docker compose up                 → postgres; migrate se preskače
docker compose --profile tools up → uključeni su i tools servisi
docker compose run --rm migrate   → eksplicitno pokreni jednokratni migrate servis
```

```text
docker compose stop    → zaustavi containere, zadrži ih
docker compose down    → ukloni containere i Compose mrežu, named volume ostaje
docker compose down -v → ukloni i volume; lokalni DB podaci se brišu
```

## Dockerfile za Go API

Dockerfile je recept iz kojeg `docker build` pravi image. MultiBook koristi **multi-stage build**:

```text
build stage   → Go compiler + source code → kompajlira `/out/api`
runtime stage → kopira samo izvršni `/api` → finalni production image
```

Najvažnije instrukcije:

```text
FROM       → izaberi osnovni image ili započni novi stage
WORKDIR    → postavi radni direktorij unutar imagea
COPY       → kopiraj fajlove iz build contexta u image
RUN        → izvrši komandu tokom pravljenja imagea
EXPOSE     → dokumentuj port na kojem aplikacija sluša
USER       → izaberi OS korisnika za pokretanje procesa
ENTRYPOINT → glavni proces koji container pokreće
```

### Zašto prvo kopiramo module fajlove?

Docker image se gradi u layerima i može ponovo koristiti nepromijenjene layere iz cachea. Ako se input neke instrukcije promijeni, taj layer i svi poslije njega moraju se ponovo izgraditi.

```dockerfile
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build ...
```

Source code se mijenja često, a dependencyji rjeđe:

```text
promijenjen samo `.go` fajl → dependency layer ostaje cached; ponavlja se COPY sourcea i build
promijenjen `go.mod`       → ponovo se izvršava `go mod download` i svi naredni layeri
```

Ako bismo prvo uradili `COPY . .`, svaka promjena source koda invalidirala bi layer i nepotrebno ponovo pokrenula download dependencyja.

Uobičajena poboljšana varijanta za Go projekte kopira oba module fajla:

```dockerfile
COPY go.mod go.sum ./
RUN go mod download
```

`go.sum` sadrži očekivane checksume verzija dependencyja i pomaže reproducibilnom, provjerenom buildu.

Build context treba ograničiti `.dockerignore` fajlom. Bez njega `COPY . .` može Dockeru poslati nepotrebne ili osjetljive lokalne fajlove poput `.git` i `.env`, čak i kada ih finalni multi-stage image ne kopira.

```text
docker build → izvršava Dockerfile i pravi image
docker run   → iz postojećeg imagea pravi i pokreće container
```

Finalni image nema Go compiler ni source code. `CGO_ENABLED=0` pravi samostalan Go binary, `GOOS=linux` cilja Linux container, a non-root runtime smanjuje privilegije procesa.

### Graceful shutdown

Graceful shutdown je kontrolisano gašenje servera: prestaje prihvatati nove requeste, daje aktivnim requestima kratko vrijeme da završe, zatvara resurse i tek onda gasi proces.

```text
Docker zaustavlja container
    ↓ šalje SIGTERM
Go `main` prima signal
    ↓ `server.Shutdown(...)`
prestaje primati nove requeste i čeka aktivne
    ↓ najviše 20 sekundi u MultiBooku
proces se uredno završava
```

Naglo gašenje može prekinuti response ili posao usred izvršavanja. `ENTRYPOINT ["/api"]` pokreće API direktno, pa Go proces pravilno prima Docker signale.

## Pokretanje Go API imagea

```text
docker build -t multibook-api:local . → Dockerfile pretvori u imenovani image
docker run ... multibook-api:local    → iz imagea napravi i pokreni container
```

Image sadrži aplikaciju, ali runtime konfiguracija dolazi tek pri pokretanju:

```text
image    → isti binary za sva okruženja
env vars → DATABASE_URL, PORT, FIREBASE_PROJECT_ID i ostala konfiguracija
mount    → Firebase credential fajl dostupan containeru bez ugrađivanja u image
port map → host:8080 prema API containeru:8080
```

`godotenv.Load()` pokušava učitati `.env`, ali u containeru `.env` nije potreban ako Docker direktno postavi environment varijable.

Ako API radi u containeru, `localhost` označava API container:

```text
API container → host.docker.internal:5432 → PostgreSQL objavljen na Macu
API u Compose networku → postgres:5432 → PostgreSQL service direktno
```

Firebase credential putanja s Maca ne vrijedi automatski unutar containera. Fajl se montira read-only na internu putanju, a `GOOGLE_APPLICATION_CREDENTIALS` pokazuje na tu internu putanju.

### Kako bi API izgledao kao Compose service?

API trenutno nije definisan u `docker-compose.yml`; lokalno se pokreće pomoću `go run`. Konceptualna Compose definicija bila bi:

```yaml
services:
  api:
    build:
      context: .
    ports:
      - "8080:8080"
    env_file:
      - .env
    environment:
      DATABASE_URL: postgres://multibook:multibook@postgres:5432/multibook?sslmode=disable
      GOOGLE_APPLICATION_CREDENTIALS: /secrets/firebase.json
    volumes:
      - /host/putanja/firebase.json:/secrets/firebase.json:ro
    depends_on:
      postgres:
        condition: service_healthy
```

```text
build.context       → koristi Dockerfile iz trenutnog direktorija
ports               → Mac:8080 prosljeđuje u API container:8080
env_file             → učitaj zajedničke runtime varijable iz `.env`
environment          → overrideuj vrijednosti koje su drugačije unutar containera
DATABASE_URL         → `postgres:5432`, jer je `postgres` Compose hostname
volumes              → montiraj Firebase JSON u container kao read-only
depends_on + healthy → pokreni API tek kada je PostgreSQL spreman
```

Pokretanje:

```text
docker compose up --build api
```

`--build` obnovi API image prije pokretanja. `depends_on` ne primjenjuje SQL migracije; one se moraju izvršiti posebno ili se mora eksplicitno povezati migration job u startup tok.
