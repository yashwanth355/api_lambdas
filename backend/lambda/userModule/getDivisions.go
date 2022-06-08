//Updated query and deployed-Sep6
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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

type divisions struct {
	Division string `json:"division"`
}

type departmentName struct {
	DepartmentName string `json:"deptName"`
}

func getDivisions(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var department departmentName
	err := json.Unmarshal([]byte(request.Body), &department)
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

	// create the select sql query
	sqlStatement := `SELECT initcap(divmaster) FROM dbo.division_master_newpg where UPPER(deptname)=$1`

	rows, err := db.Query(sqlStatement, strings.ToUpper(department.DepartmentName))

	var divisionNames []divisions
	defer rows.Close()
	for rows.Next() {
		var divisionName divisions
		err = rows.Scan(&divisionName.Division)
		divisionNames = append(divisionNames, divisionName)
	}
	res, _ := json.Marshal(divisionNames)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getDivisions)
}
