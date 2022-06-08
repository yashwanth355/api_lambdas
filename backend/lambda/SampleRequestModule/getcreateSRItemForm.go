// CHECKEDIN
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"
)

const (
	host     = "ccl-psql-dev.cclxlbtddgmn.ap-south-1.rds.amazonaws.com"
	port     = 5432
	user     = "postgres"
	password = "Ccl_RDS_DB#2022"
	dbname   = "ccldevdb"
)

type SRItemForm struct {
	ProductType    []ProductType    `json:"product_type"`
	SampleCategory []SampleCategory `json:"sample_category"`
}
type ProductType struct {
	ProductID   int    `json:"product_id"`
	ProductName string `json:"product_name"`
}
type SampleCategory struct {
	SamplecatID  int    `json:"samplecat_id"`
	CategoryName string `json:"sample_category"`
}

func getcreateSRItemForm(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Hefiers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	defer db.Close()

	// check db
	err = db.Ping()

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	fmt.Println("Connected!")

	var fi SRItemForm
	var res []byte

	var rows *sql.Rows
	log.Println("get all product names")
	sqlStatement1 := `select productid,initcap(productname) as name FROM dbo.prod_product_master
	ORDER BY productid ASC`

	rows, err = db.Query(sqlStatement1)

	defer rows.Close()
	for rows.Next() {
		var pt ProductType
		err = rows.Scan(&pt.ProductID, &pt.ProductName)
		allProducts := append(fi.ProductType, pt)
		fi.ProductType = allProducts
	}

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	log.Println("get all Category names")
	sqlStatement2 := `SELECT samplecatid,sample_category FROM dbo.cms_sample_category_master
					ORDER BY samplecatid ASC`

	rows, err = db.Query(sqlStatement2)

	defer rows.Close()
	for rows.Next() {
		var sc SampleCategory
		err = rows.Scan(&sc.SamplecatID, &sc.CategoryName)
		allCats := append(fi.SampleCategory, sc)
		fi.SampleCategory = allCats
	}

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	res, _ = json.Marshal(fi)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getcreateSRItemForm)
}
