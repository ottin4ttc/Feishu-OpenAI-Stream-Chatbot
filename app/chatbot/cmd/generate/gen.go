package main

import (
	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gorm"
)

func main() {
	// Initialize the generator with configuration
	g := gen.NewGenerator(gen.Config{
		OutPath:       "./dal/dsDb/query", // output directory, default value is ./query
		ModelPkgPath:  "./model",
		Mode:          gen.WithDefaultQuery,
		FieldNullable: false,
	})

	// Initialize a *gorm.DB instance
	db, _ := gorm.Open(postgres.Open("host=71.132.58.117 user=core password=ottinPassword dbname=core port=5432 sslmode=disable"), &gorm.Config{})

	// Use the above `*gorm.DB` instance to initialize the generator,
	// which is required to generate structs from userdb when using `GenerateModel/GenerateModelAs`
	g.UseDB(db)

	// Generate default DAO interface for those generated structs from database
	g.ApplyBasic(
		g.GenerateModel("ds_message"),
	)

	// Execute the generator
	g.Execute()
}
