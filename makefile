postgres:
	docker run --name postgres --network bank-network -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=secret -d postgres:14-alpine

createdb:
	docker exec -it postgres createdb --username=postgres --owner=postgres simple_bank

dropdb:
	docker exec -it postgres dropdb simple_bank

# video 5
migrateup:
	migrate -path db/migration -database "postgresql://postgres:secret@localhost:5432/gobank?sslmode=disable" -verbose up
#  migrate -path db/migration -database "$(DB_URL)" -verbose up

migratedown:
	migrate -path db/migration -database "postgresql://postgres:secret@localhost:5432/gobank?sslmode=disable" -verbose down
# migrate -path db/migration -database "$(DB_URL)" -verbose down	

sqlc:
	sqlc generate

test:
	go test -v -cover ./...	

mysql:
	docker run --name mysql8 -p 3306:3306  -e MYSQL_ROOT_PASSWORD=secret -d mysql:8

server:
	 go run main.go

mock: 
	mockgen -package mockdb -destination db/mock/store.go GoBank/db/sqlc Store


.PHONY: postgres createdb dropdb migrateup migratedown sqlc test server mock

