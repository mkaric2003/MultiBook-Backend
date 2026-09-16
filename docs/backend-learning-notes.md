# Backend bilješke

Kratke bilješke nastale tokom učenja backenda na MultiBook projektu.

## MultiBook sistem

```text
Flutter aplikacija
    ↓ HTTP/JSON + Firebase token
Go API
    ↓ SQL
PostgreSQL
```

- **Flutter** – korisnički interfejs i slanje zahtjeva.
- **Go API** – validira zahtjeve, provodi poslovna pravila i kontroliše pristup podacima.
- **PostgreSQL / Supabase** – trajno čuva poslovne podatke.
- **Firebase Authentication** – potvrđuje identitet korisnika.
- **Firebase Storage** – čuva slike i fajlove.
- **Firebase Cloud Messaging** – šalje push notifikacije.
- **Docker** – lokalno pokreće izolovan PostgreSQL i druge potrebne alate.

Backend je autoritet: ne vjeruje podacima koje klijent može izmijeniti i sam provjerava identitet, dozvole i poslovna pravila.

MultiBook backend je **modularni monolit**: jedan Go proces, podijeljen na odvojene module kao što su `users`, `businesses`, `bookings` i `appointments`.

## HTTP

HTTP je dogovor po kojem klijent i server razmjenjuju zahtjeve i odgovore.

### Request

```text
HTTP metoda + putanja + headeri + opcionalni body
```

Primjer:

```http
GET /v1/users/me
Authorization: Bearer <firebase_token>
```

- **Metoda** – govori šta želimo uraditi (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`).
- **Putanja** – govori nad kojim resursom radimo.
- **Headeri** – nose dodatne informacije, npr. token ili format podataka.
- **Body** – nosi podatke za kreiranje ili izmjenu; `GET` ga obično nema.

### Response

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "...",
  "email": "...",
  "role": "customer"
}
```

- **Status code** – govori kako je zahtjev završio.
- **Response headeri** – opisuju odgovor.
- **Response body** – sadrži vraćene podatke ili opis greške.

Najčešći statusi:

```text
200 → zahtjev uspješan
201 → resurs kreiran
204 → uspješno, bez response bodyja
401 → korisnik nije autentifikovan
403 → korisnik nema dozvolu
404 → resurs nije pronađen
422 → poslani podaci nisu validni
429 → poslano je previše zahtjeva
500 → neočekivana serverska greška
```

## Resursi i putanje

Resurs je podatak ili skup podataka kojim backend upravlja.

```text
GET    /businesses          → pročitaj kolekciju biznisa
POST   /businesses          → kreiraj novi biznis
GET    /businesses/{id}     → pročitaj konkretan biznis
PATCH  /businesses/{id}     → izmijeni konkretan biznis
DELETE /businesses/{id}     → ukloni ili arhiviraj konkretan biznis
```

```text
Putanja         → bira vrstu ili identitet resursa
HTTP metoda     → bira operaciju nad resursom
Query parametri → filtriranje, sortiranje ili paginacija
Body            → podaci za kreiranje ili izmjenu
```

Primjer:

```http
GET /v1/businesses?city=Sarajevo
```

`/businesses` bira kolekciju, a `city=Sarajevo` je filtrira.

## Router i handler

Router povezuje kombinaciju HTTP metode i putanje s handler funkcijom.

```go
r.Get("/users/me", usersHandler.CurrentUser)
```

```text
GET /v1/users/me → CurrentUser handler
```

Handler prima:

```text
*http.Request       → ono što je klijent poslao
http.ResponseWriter → alat za pravljenje odgovora klijentu
```

## Middleware

Middleware je kod koji se izvršava prije ili poslije handlera. Koristi se za zajedničke poslove koji važe za više ruta.

```text
Request → middleware → middleware → handler → response
```

Middleware poziva:

```go
next.ServeHTTP(w, r)
```

da bi zahtjev pustio dalje. Ako vrati odgovor i uradi `return` bez pozivanja `next`, lanac se zaustavlja.

### MultiBook middleware lanac

```text
RequestID      → označi zahtjev jedinstvenim ID-jem
RealIP         → pronađi stvarnu IP adresu klijenta
Recoverer      → uhvati neočekivani panic i vrati kontrolisanu grešku
RequestLogger  → zapiši metodu, putanju, status i trajanje
CORS           → kontroliši koji browser origin smije pozvati API
RateLimit      → ograniči broj zahtjeva po IP adresi
RequestTimeout → ograniči trajanje običnog zahtjeva
Auth           → potvrdi Firebase identitet korisnika
ProvisionUser  → učitaj MultiBook korisnika
Handler        → obradi konkretnu rutu
```

Globalni middlewarei se registruju pomoću `router.Use(...)` i djeluju na sve rute tog routera.

Middlewarei registrovani unutar `/v1` grupe djeluju samo na `/v1` rute:

```go
router.Route("/v1", func(r chi.Router) {
    r.Use(authenticator.Require)
    r.Use(usershttp.ProvisionUser(usersService))
})
```

`/healthz` i `/readyz` su izvan ove grupe, pa ne zahtijevaju Firebase token.

## Autentifikacija

Flutter šalje Firebase ID token:

```http
Authorization: Bearer <firebase_token>
```

Backend provjerava token preko Firebase Admin SDK-a. Ako token nedostaje, nije validan, istekao je ili je opozvan, backend vraća `401 Unauthorized` i ne poziva handler.

Provjereni token sprema se u **request context**.

Request context je prostor vezan samo za jedan HTTP zahtjev kroz koji middlewarei i handler mogu prenositi podatke, poput Firebase tokena i trenutnog korisnika.

### Trajno čuvanje naspram request contexta

```text
PostgreSQL users tabela → trajno čuva MultiBook korisnika
Request context         → privremeno prenosi korisnika kroz jedan request
```

Context nije baza, cache niti korisnička sesija. Postoji samo tokom obrade konkretnog requesta i nakon završetka se više ne koristi.

Go context služi za:

- prenošenje request-scoped podataka, npr. tokena i trenutnog korisnika;
- signal da je klijent prekinuo request;
- timeout i otkazivanje rada kroz sve slojeve.

Context se prosljeđuje kroz lanac:

```text
HTTP request → middleware → handler → service → PostgreSQL adapter
```

## Provisioning korisnika

Firebase korisnik predstavlja identitet, dok PostgreSQL `users` zapis sadrži MultiBook profil i poslovne podatke.

`ProvisionUser` middleware povezuje ta dva korisnika:

```text
Provjeren Firebase token
    ↓ UID, email i display name
Application service
    ↓ kreiraj zapis ako ne postoji
PostgreSQL users tabela
    ↓ učitaj trenutni zapis
domain.User u request contextu
```

Provisioning koristi idempotentni SQL:

```sql
INSERT INTO users (...)
ON CONFLICT (id) DO NOTHING
```

- Prvi zahtjev korisnika → kreira `users` zapis.
- Svaki sljedeći zahtjev → postojeći zapis ostaje nepromijenjen.
- **Idempotentna operacija** → ponavljanje ima isti konačni rezultat kao jedno izvršavanje.

U trenutnoj implementaciji provisioning `INSERT` se pokušava na svakom autentifikovanom `/v1` requestu. To osigurava da zapis postoji bez posebnog registration endpointa. `ON CONFLICT DO NOTHING` čini ponovljeni poziv sigurnim, ali i dalje predstavlja odlazak do baze. Jedan `UPSERT ... RETURNING` bio bi moguća optimizacija kojom bi se kreiranje i učitavanje spojili u jedan upit.

Nakon provisioninga, korisnik se učitava iz baze i stavlja u request context kako bi ga handleri i servisi mogli koristiti.

## Pristup PostgreSQL bazi

MultiBook koristi `pgx` Go driver i `pgxpool.Pool` za upravljanje konekcijama prema PostgreSQL-u.

```text
Go request → uzme slobodnu konekciju iz poola → izvrši SQL → vrati konekciju u pool
```

- **Pool** – skup već otvorenih konekcija koje requesti ponovo koriste.
- **Exec** – izvršava SQL kada nam redovi rezultata nisu potrebni.
- **QueryRow** – izvršava upit od kojeg očekujemo jedan red.
- **Scan** – kopira kolone rezultata u Go varijable ili polja structa.

PostgreSQL parametri koriste `$1`, `$2`, `$3`:

```go
pool.Exec(ctx,
    `INSERT INTO users (id, email) VALUES ($1, $2)`,
    userID,
    email,
)
```

```text
$1 → userID
$2 → email
```

Vrijednosti se šalju odvojeno od SQL teksta. Driver ih sigurno prosljeđuje bazi, što štiti od SQL injectiona.

```text
QueryRow → PostgreSQL pronađe red → Scan → domain.User
```

## PostgreSQL tabela

```text
Tabela  → kolekcija podataka iste vrste
Red     → jedan zapis, npr. jedan korisnik
Kolona  → jedno svojstvo, npr. email ili role
Tip     → ograničava vrstu vrijednosti u koloni
```

Važni SQL pojmovi:

- `PRIMARY KEY` – jedinstveno identifikuje svaki red; automatski je unique i indeksiran.
- `NULL` – vrijednost nije poznata ili nije postavljena; nije isto što i prazan string.
- `NOT NULL` – kolona mora imati vrijednost.
- `DEFAULT` – vrijednost koju baza postavlja ako nije poslana.
- `CHECK` – pravilo kojim baza odbija nedozvoljenu vrijednost.
- `FOREIGN KEY` – veza prema redu druge tabele.
- `INDEX` – dodatna struktura koja ubrzava određena čitanja uz cijenu prostora i nešto sporijih upisa.

### Foreign key

```sql
FOREIGN KEY (selected_business_id)
REFERENCES businesses(id)
ON DELETE SET NULL
```

```text
users.selected_business_id → businesses.id
```

Foreign key osigurava **referencijalni integritet**: veza ne smije pokazivati na red koji ne postoji.

```text
users.selected_business_id = NULL   → dozvoljeno; korisnik nema odabrani biznis
users.selected_business_id = biz-10 → dozvoljeno samo ako businesses.id = biz-10 postoji
users.selected_business_id = biz-99 → baza odbija upis ako biz-99 ne postoji
```

Bez foreign keya baza bi prihvatila `biz-99`, pa bismo imali pokvarenu ili "viseću" referencu. `ON DELETE SET NULL` dodatno određuje da se `selected_business_id` postavi na `NULL` ako se povezani business fizički obriše.

Foreign key ne znači da je korisnik vlasnik biznisa; vlasništvo je posebno poslovno pravilo.

Stvarni primjer iz `stay_bookings`:

```sql
business_id       UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
business_owner_id TEXT NOT NULL REFERENCES users(id)      ON DELETE RESTRICT,
customer_id       TEXT NOT NULL REFERENCES users(id)      ON DELETE RESTRICT
```

Pri `INSERT`-u rezervacije PostgreSQL automatski provjerava da poslani business, owner i customer postoje. `ON DELETE RESTRICT` zabranjuje fizičko brisanje roditeljskog reda dok ga rezervacija referencira, čime se čuva historija rezervacije. Foreign key se ne poziva ručno iz Go koda; baza ga automatski primjenjuje na svaki `INSERT`, promjenu FK vrijednosti i fizički `DELETE` povezanog reda.

### Indeksiranje kolone

```sql
CREATE INDEX users_email_idx ON users (email)
WHERE deleted_at IS NULL;
```

PostgreSQL kreira odvojenu, uređenu strukturu koja povezuje vrijednost kolone s lokacijom reda. Ovo je partial index jer sadrži samo aktivne redove.

```text
email vrijednost → lokacija odgovarajućeg reda
```

Indeks najviše pomaže upitima koji filtriraju, spajaju ili sortiraju po indeksiranim kolonama. `PRIMARY KEY` automatski dobija unique indeks, ali PostgreSQL ne kreira automatski indeks na koloni koja sadrži foreign key.

### Soft delete

```text
Hard delete → red se fizički briše
Soft delete → red ostaje, ali `deleted_at` dobija vrijeme brisanja
```

Aktivni korisnici se čitaju uslovom:

```sql
WHERE deleted_at IS NULL
```

## JOIN

`JOIN` spaja povezane redove iz dvije ili više tabela u jedan rezultat.

```sql
SELECT u.id, u.email, b.name
FROM users u
JOIN businesses b ON b.id = u.selected_business_id;
```

- `u` i `b` su kratki aliasi za tabele.
- `ON` definiše uslov povezivanja redova.
- `JOIN` / `INNER JOIN` vraća samo redove koji imaju par u obje tabele.
- `LEFT JOIN` vraća sve redove lijeve tabele; ako nema para desno, desne kolone su `NULL`.

```text
Foreign key → sprečava da sačuvamo referencu prema nepostojećem redu
JOIN        → koristi vrijednosti veze da sastavi podatke iz više tabela
```

## Migracije baze

Migracija je verzionisana SQL promjena strukture ili podataka baze. Fajlovi se izvršavaju po broju:

```text
000001 → 000002 → 000003 → ...
```

Svaka verzija obično ima par:

```text
000011_service_appointments.up.sql   → primijeni promjenu
000011_service_appointments.down.sql → poništi promjenu
```

- `up` može kreirati ili mijenjati tabele, kolone, foreign keyeve i indekse.
- `down` vraća bazu jedan korak unazad.
- Migration alat u `schema_migrations` pamti primijenjenu verziju i je li izvršavanje ostalo u `dirty` stanju.
- Već dijeljenu migraciju ne prepravljamo; pravimo novu migraciju s narednim brojem.

### schema_migrations

`schema_migrations` je interna tabela migration alata, a ne MultiBook poslovna tabela. U njoj alat prati:

```text
version → broj trenutno primijenjene migracije
dirty   → je li posljednja migracija prekinuta ili ostala nedovršena
```

Ako je `version = 15`, sljedeći `migrate up` počinje od `000016`. Vrijednosti ne mijenjamo ručno; `force` se koristi samo nakon provjere i ručnog popravljanja stvarne strukture baze.

Lokalno:

```text
make migrate-up   → pokreni PostgreSQL i primijeni sve nove `up` migracije
make migrate-down → poništi posljednju migraciju
```

## Docker

Docker bilješke su izdvojene u poseban dokument: [Docker bilješke](./docker-learning-notes.md).

## PATCH request i JSON decoding

```text
POST  → kreiraj novi resurs u kolekciji
PUT   → postavi ili potpuno zamijeni stanje resursa
PATCH → djelimično izmijeni samo poslana polja
```

MultiBook profil koristi:

```http
PATCH /v1/users/me
Content-Type: application/json

{"city":"Sarajevo"}
```

Handler dekodira JSON u Go struct:

```go
var input domain.UpdateProfileInput
decodeJSON(w, r, &input)
```

- `json:"first_name"` mapira JSON `first_name` na Go `FirstName`.
- `&input` daje decoderu adresu structa kako bi ga mogao popuniti.
- Pointer polje `*string` je `nil` kada nije poslano; SQL tada zadržava postojeću vrijednost.
- U trenutnom modelu i izostavljeno polje i eksplicitni JSON `null` završavaju kao `nil`, pa se njima vrijednost ne može obrisati.

Zajednički `DecodeJSON`:

```text
ograničava body na 1 MiB
odbija nepoznata JSON polja
zahtijeva tačno jednu JSON vrijednost
```

Neispravan body vraća `422 validation_error`. ID trenutnog korisnika uzima se iz provjerenog Firebase tokena, nikada iz request bodyja.
