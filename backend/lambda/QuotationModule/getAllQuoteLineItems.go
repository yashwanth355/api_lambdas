package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

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

type QuoteLineItem struct {
	QuoteLineItemId     string `json:"quotelineitem_id"`
	QuoteId             int    `json:"quote_id"`
	QuoteLineItemNumber string `json:"quotelineitem_number"`
	SampleCode          string `json:"sample_code"`
	ExpectedOrder       int    `json:"expectedorder_kgs"`
	Category            string `json:"category"`
	CategoryTypeId      int    `json:"categorytype_id"`
	WeightId            int    `json:"weight_id"`
	CategoryType        string `json:"categorytype"`
	Weight              string `json:"weight"`
	UPCId               int    `json:"upc"`
	UPC                 string `json:"upc_id"`
	BasePrice           string `json:"baseprice"`
	FinalPrice          string `json:"final_price"`
	CustApprove         string `json:"customer_approval"`
	GmsApprovalStatus   string `json:"gms_approvalstatus"`
}

type Input struct {
	QuoteId int `json:"quote_id"`
}

func getAllQuoteLineItems(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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

	rows, err := db.Query(``)

	sqlStatement := `select q.quoteitemid,q.lineitemnumber,q.quoteid,h.samplecode,q.expectedorderqty, s.categoryname,q.packcategorytypeid,q.packweightid, q.packupcid,
	q.basepricekgs,
	q.finalprice,
	q.custapprove,
	q.gmsapprovalstatus
	from dbo.cms_quote_item_master q
	LEFT JOIN dbo.cms_prod_pack_category s on q.packcategoryid = s.categoryid
	INNER JOIN dbo.qua_product_sample_master h on q.sampleid = h.sampleid
	where q.quoteid=$1 order by q.createddate desc`
	rows, err = db.Query(sqlStatement, input.QuoteId)
	if err != nil {
		log.Println(err)
		log.Println(err, "Not able to get information from quote line item table")
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	var category, packCategoryTypeId, packWeightId, packupcId, basePrice, finalPrice, custApprovalStatus, gmsApprovalStatus sql.NullString

	var allQuoteLineItems []QuoteLineItem
	defer rows.Close()
	for rows.Next() {
		var lineItem QuoteLineItem
		err = rows.Scan(&lineItem.QuoteLineItemId, &lineItem.QuoteLineItemNumber,
			&lineItem.QuoteId,
			&lineItem.SampleCode,
			&lineItem.ExpectedOrder,
			&category,
			&packCategoryTypeId,
			&packWeightId,
			&packupcId,
			&basePrice,
			&finalPrice,
			&custApprovalStatus,
			&gmsApprovalStatus)

		lineItem.CategoryTypeId, _ = strconv.Atoi(packCategoryTypeId.String)
		lineItem.WeightId, _ = strconv.Atoi(packWeightId.String)
		lineItem.UPCId, _ = strconv.Atoi(packupcId.String)
		lineItem.Category = category.String
		lineItem.BasePrice = basePrice.String
		lineItem.FinalPrice = finalPrice.String
		lineItem.GmsApprovalStatus = gmsApprovalStatus.String
		custApprove, _ := strconv.Atoi(custApprovalStatus.String)

		if custApprove == 1 {
			lineItem.CustApprove = "Approved"
		} else if custApprove == 0 && custApprovalStatus.String != "" {
			lineItem.CustApprove = "Rejected"
		}

		if lineItem.CategoryTypeId != 0 {

			sqlStatement := `select categorytypename from dbo.cms_prod_pack_category_type where categorytypeid=$1`
			rows1, err := db.Query(sqlStatement, lineItem.CategoryTypeId)

			if err != nil {
				log.Println(err)
				log.Println("unable to add categorytype", lineItem.CategoryTypeId)
			}

			for rows1.Next() {
				err = rows1.Scan(&lineItem.CategoryType)
			}

		}

		if lineItem.WeightId != 0 {

			sqlStatement := `select weightname from dbo.cms_prod_pack_category_weight where weightid=$1`
			rows2, err := db.Query(sqlStatement, lineItem.WeightId)

			if err != nil {
				log.Println(err)
				log.Println("unable to add weight", lineItem.WeightId)
			}

			for rows2.Next() {
				err = rows2.Scan(&lineItem.Weight)
			}

		}

		if lineItem.UPCId != 0 {
			sqlStatement := `select upcname from dbo.cms_prod_pack_upc where upcid=$1`
			rows5, err := db.Query(sqlStatement, lineItem.UPCId)

			if err != nil {
				log.Println(err)
			}

			for rows5.Next() {
				err = rows5.Scan(&lineItem.UPC)
			}
		}
		allQuoteLineItems = append(allQuoteLineItems, lineItem)
	}
	res, _ := json.Marshal(allQuoteLineItems)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getAllQuoteLineItems)
}
