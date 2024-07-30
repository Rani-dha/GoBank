postgres:
	docker run --name postgres --network bank-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:14-alpine

createdb:
	docker exec -it postgres createdb --username=root --owner=root simple_bank

dropdb:
	docker exec -it postgres dropdb simple_bank

# video 5
migrateup:
	migrate -path db/migration -database "postgresql://postgres:secret@localhost:5432/simplebank?sslmode=disable" -verbose up
#  migrate -path db/migration -database "$(DB_URL)" -verbose up

migratedown:
	migrate -path db/migration -database "postgresql://postgres:secret@localhost:5432/simplebank?sslmode=disable" -verbose down

# migrate -path db/migration -database "$(DB_URL)" -verbose down	


sqlc:
	sqlc generate



.PHONY: postgres createdb dropdb migrateup migratedown sqlc

