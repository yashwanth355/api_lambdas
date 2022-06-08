package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
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

type UserDetails struct {
	LastIdsno         int    `json:"lastidsno"`
	Idsno             int    `json:"idsno"`
	Update            bool   `json:"update"`
	Userid            string `json:"userid"`
	Firstname         string `json:"firstname"`
	Middlename        string `json:"middlename"`
	Lastname          string `json:"lastname"`
	Emailid           string `json:"emailid"`
	Alias             string `json:"alias"`
	Username          string `json:"username"`
	Empcode           string `json:"empcode"`
	Designation       string `json:"designation"`
	Company           string `json:"company"`
	Department        string `json:"department"`
	Ext               string `json:"ext"`
	Mobile            string `json:"mobile"`
	Phone             string `json:"phone"`
	Role              string `json:"role"`
	Division          string `json:"division"`
	Employee          bool   `json:"employee"`
	Profile           string `json:"profile"`
	Title             string `json:"title"`
	Active            bool   `json:"active"`
	Delegatedapprover string `json:"delegatedapprover"`
	Manager           string `json:"manager"`
	Street            string `json:"street"`
	State             string `json:"state"`
	Postalcode        string `json:"postalcode"`
	City              string `json:"city"`
	Country           string `json:"country"`
	Password          string `json:"password"`
}

func insertUserDetails(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var user UserDetails
	err := json.Unmarshal([]byte(request.Body), &user)
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

	if user.Update {

		log.Println("entered into update")
		var tt int
		sqlStatementDuplicateEmail := `select count(emailid) from dbo.users_master_newpg where UPPER(emailid) = $1`
		rows1, err := db.Query(sqlStatementDuplicateEmail, strings.ToUpper(user.Emailid))

		log.Println("before the call")

		// var vendor InputAdditionalDetails
		for rows1.Next() {
			err = rows1.Scan(&tt)
		}
		if err != nil {
			log.Println(err.Error())
		}
		log.Println("email id validation check")

		if tt > 1 {
			log.Println("Email id is already present")
			return events.APIGatewayProxyResponse{230, headers, nil, "User already exist with same email", false}, nil
		}
		sqlStatement1 := `UPDATE dbo.users_master_newpg SET 
			firstname =$1,
			middlename =$2,
			lastname =$3,
			alias =$4,
			username =$5,
			empcode =$6,
			designation =$7,
			company=$8,
			department=$9,
			role=$10,
			division=$11,
			employee=$12,
			profile=$13,
			delegatedapprover =$14,
			manager=$15,
			active=$16,
			street =$17,
			state=$18,
			postalcode=$19,
			title=$20,
			city=$21,
			country=$22,
			ext=$23,
			phone=$24,
			mobile=$25 where emailid=$26`

		_, err = db.Query(sqlStatement1,
			user.Firstname, user.Middlename, user.Lastname, user.Alias, user.Username, user.Empcode, user.Designation, user.Company, user.Department, user.Role, user.Division, user.Employee, user.Profile, user.Delegatedapprover, user.Manager, user.Active, user.Street, user.State, user.Postalcode, user.Title, user.City, user.Country, user.Ext, user.Phone, user.Mobile, user.Emailid)

		if err != nil {
			log.Println(err.Error())
			log.Println("unable to update user details")
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		return events.APIGatewayProxyResponse{200, headers, nil, string("Updated successfully"), false}, nil

	} else {

		log.Println("entered into insert")
		var tt string
		sqlStatementDuplicateEmail := `select emailid from dbo.users_master_newpg where UPPER(emailid) = $1`
		rows1, err := db.Query(sqlStatementDuplicateEmail, strings.ToUpper(user.Emailid))

		log.Println("before the call")

		// var vendor InputAdditionalDetails
		for rows1.Next() {
			err = rows1.Scan(&tt)
		}
		if err != nil {
			log.Println(err.Error())

		}
		log.Println(tt + "email id validation check")

		if tt != "" {
			log.Println("Email id is already present")
			return events.APIGatewayProxyResponse{230, headers, nil, "User already exist with same email", false}, nil
		}
		//Find latest idsno
		sqlStatementUID1 := `SELECT idsno 
							FROM dbo.users_master_newpg 
							where idsno is not null
							ORDER BY idsno DESC 
							LIMIT 1`
		rows, err = db.Query(sqlStatementUID1)
		if err != nil {
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		for rows.Next() {
			err = rows.Scan(&user.LastIdsno)
		}
		log.Println("Last Idsno: ", user.LastIdsno)
		//Generating UserIds NOs----------------
		user.Idsno = user.LastIdsno + 1
		user.Userid = strconv.Itoa(user.Idsno)
		log.Println("New Idsno: ", user.Idsno)
		log.Println("New Userid : ", user.Userid)

		sqlStatementIU1 := `INSERT INTO dbo.users_master_newpg(idsno,userid,firstname, middlename, lastname, emailid, alias, username, empcode, designation, company, department, role, division, employee, profile, delegatedapprover, manager, active, street, state, postalcode, title, city, country, password, ext, phone, mobile) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28,$29)`

		_, err1 := db.Query(sqlStatementIU1, user.Idsno, user.Userid, user.Firstname, user.Middlename, user.Lastname, user.Emailid, user.Alias, user.Username, user.Empcode, user.Designation, user.Company, user.Department, user.Role, user.Division, user.Employee, user.Profile, user.Delegatedapprover, user.Manager, user.Active, user.Street, user.State, user.Postalcode, user.Title, user.City, user.Country, user.Password, user.Ext, user.Phone, user.Mobile)

		if err1 != nil {
			log.Println(err1.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err1.Error(), false}, nil
		}
		return events.APIGatewayProxyResponse{200, headers, nil, string("Created user successfully"), false}, nil
	}
}

func main() {
	lambda.Start(insertUserDetails)
}
