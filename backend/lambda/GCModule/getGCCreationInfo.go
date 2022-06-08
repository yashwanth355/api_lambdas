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

type UOMSList struct {
	UomId   string `json:"uom_id"`
	UomName string `json:"uom_name"`
}

type ItemCategoryList struct {
	CategoryId   string `json:"item_category_id"`
	CategoryName string `json:"item_category_name"`
}

type ItemGroupsList struct {
	GroupId    string `json:"group_id"`
	GroupName  string `json:"group_name"`
	LedGroupId string `json:"led_group_id"`
}

type Input struct {
	Type string `json:"type"`
}

func getGCCreationInfo(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var input Input
	err := json.Unmarshal([]byte(request.Body), &input)
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

	var res []byte
	var rows *sql.Rows
	if input.Type == "uoms" {
		log.Println("get all uoms", input.Type)
		sqlStatement := `SELECT uom, initcap(uomname) FROM dbo.project_uom
		ORDER BY uom ASC`
		rows, err = db.Query(sqlStatement)

		var allUomsList []UOMSList
		defer rows.Close()
		for rows.Next() {
			var ct UOMSList
			err = rows.Scan(&ct.UomId, &ct.UomName)
			allUomsList = append(allUomsList, ct)
		}

		res, _ = json.Marshal(allUomsList)
	} else if input.Type == "itemcategories" {
		log.Println("get all item categories", input.Type)
		sqlStatement := `select itemcatid, initcap(itemcatname) from dbo.INV_ITEM_CATEGORY`
		rows, err = db.Query(sqlStatement)

		var allItemCategoryList []ItemCategoryList
		defer rows.Close()
		for rows.Next() {
			var ct ItemCategoryList
			err = rows.Scan(&ct.CategoryId, &ct.CategoryName)
			allItemCategoryList = append(allItemCategoryList, ct)
		}

		res, _ = json.Marshal(allItemCategoryList)
	} else if input.Type == "itemgroups" {
		log.Println("get all item groups", input.Type)
		sqlStatement := `SELECT groupid, initcap(groupname),ledgroupid FROM dbo.inv_gc_itemgroups
		ORDER BY groupid ASC`
		rows, err = db.Query(sqlStatement)

		var allItemGroupsList []ItemGroupsList
		defer rows.Close()
		for rows.Next() {
			var ct ItemGroupsList
			err = rows.Scan(&ct.GroupId, &ct.GroupName, &ct.LedGroupId)
			allItemGroupsList = append(allItemGroupsList, ct)
		}

		res, _ = json.Marshal(allItemGroupsList)
	}

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getGCCreationInfo)
}
