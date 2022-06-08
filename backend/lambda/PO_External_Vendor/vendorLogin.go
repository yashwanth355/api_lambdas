package main

import (
	
	// "os"
	// "github.com/aws/aws-lambda-go/events"
	// "github.com/aws/aws-lambda-go/lambda"
	// "github.com/aws/aws-sdk-go/aws"
	// "github.com/aws/aws-sdk-go/aws/session"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	// "strconv"
	// "time"

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


type Credentials struct {
	Vendor	  bool 	 `json:"vendor"`
	FUsername string `json:"username"`
	FPassword string `json:"password"`
	DUsername string `json:"dusername"`
	DPassword string `json:"dpassword"`
}
type Result struct {
    Id       string `json:"id"`
    Role     string  `json:"role"`
    UserName string  `json:"user_name"`
	UserId	string  `json:"user_id"`
    
}



func vendorLogin(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var c Credentials
	var res Result
	

	err := json.Unmarshal([]byte(request.Body), &c)
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
	var rows *sql.Rows

	if (c.Vendor) && (c.FUsername != "") && (c.FPassword!="") {
		log.Println("Entered Vendor Login Module")
		sqlStatementCUser1 := `select vendorid,lower(email),password 
								from dbo.pur_vendor_master_newpg
								where
								lower(email)=$1
								and
								password=$2`
		rows, err = db.Query(sqlStatementCUser1, c.FUsername,c.FPassword)
		for rows.Next() {
			err = rows.Scan(&res.UserId,&c.DUsername,&c.DPassword)
		}
		if err != nil {
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		if c.DUsername==c.FUsername && c.DPassword==c.FPassword {
			res.Id=res.UserId
    		res.Role="Vendor"  
    		res.UserName=c.FUsername
    		  
			res, _ := json.Marshal(res)
			return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
			
		} else {
			return events.APIGatewayProxyResponse{500, headers, nil, string("Invalid Credentials"), false}, nil
		}
	
		return events.APIGatewayProxyResponse{200, headers, nil, string("success"), false}, nil
	} else {
		return events.APIGatewayProxyResponse{500, headers, nil, string("Missing Credentials"), false}, nil
	}
}

func main() {
	lambda.Start(vendorLogin)
}