//CHECKEDIN
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"bytes"
	"net/smtp"
	"text/template"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"
	"strconv"
)

const (
	host     = "ccl-psql-dev.cclxlbtddgmn.ap-south-1.rds.amazonaws.com"
	port     = 5432
	user     = "postgres"
	password = "Ccl_RDS_DB#2022"
	dbname   = "ccldevdb"
	//email-SMTP
	from_email = "itsupport@continental.coffee"
	smtp_pass = "is@98765"
	// smtp server configuration.
	smtpHost = "smtp.gmail.com"
	smtpPort = "587"
)
const QuoteTemp=`<!DOCTYPE html>
	    <html>
		<head>
			<img src="https://s3.ap-south-1.amazonaws.com/beta-a2z.cclproducts.com/static/media/CCLEmailTemplate.png">
		</head>
		<body>
			<h3>Hello,</h3>
			<p>{{.EMessage}}</p>
			
		</body>
	</html>`

type Input struct {
	Type                     string `json:"type"`
	LineItemId               int    `json:"lineitem_id"`
	Margin                   string `json:"margin"`
	MarginPercentage         string `json:"margin_percentage"`
	FinalPrice               string `json:"final_price"`
	CustApprove              int    `json:"customer_approval"`
	GmsApprovalStatus        string `json:"gms_approvalstatus"`
	GmsRejectionRemarks      string `json:"gms_rejectionremarks"`
	CustomerRejectionRemarks string `json:"customer_rejectionremarks"`
	ConfirmedOrderQuantity   string `json:"confirmed_orderquantity"`
	NegativeMarginStatus     string `json:"negativemargin_status"`
	NegativeMarginRemarks    string `json:"negativemargin_remarks"`
	NegativeMarginReason     string `json:"negativemargin_reason"`
	LoggedInUserID     		 string `json:"loginuserid"`
	Role 					string `json:"role"`
}
// Email is input request data which is used for sending email using aws ses service
type Email struct {
	
	ToEmail    string `json:"to_email"`
	ToName    string `json:"name"`
	Subject   string `json:"subject"`
	Message   string `json:"message"`
}
func NewNullString(s string) sql.NullString {
	if len(s) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{
		String: s,
		Valid: true,
	}
}

func updateQuoteLineInfo(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var input Input
	var e Email
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
	// Find created user username
	var email string
	

	log.Println("update Quoteline item")
	if input.Type == "marginInfo" {
		sqlStatement1 := `UPDATE dbo.cms_quote_item_master SET 
	margin=$1,
	marginpercentage=$2,
	finalprice=$3 where quoteitemid=$4`

		_, err = db.Query(sqlStatement1,
			input.Margin,
			input.MarginPercentage,
			input.FinalPrice,
			input.LineItemId)
		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		return events.APIGatewayProxyResponse{200, headers, nil, string("success"), false}, nil
	} else if input.Type == "customerInfo" {
		log.Println("Entered customer approval segment for the quoteline item")
		sqlStatement1 := `UPDATE dbo.cms_quote_item_master SET 
							custapprove=$1,
							rejectionremarks=$2,
							confirmedorderquantity_kgs=$3 
							where 
							quoteitemid=$4`

		_, err = db.Query(sqlStatement1,
							strconv.Itoa(input.CustApprove),
							input.CustomerRejectionRemarks,
							NewNullString(input.ConfirmedOrderQuantity), 
							input.LineItemId)
		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		return events.APIGatewayProxyResponse{200, headers, nil, string("success"), false}, nil
	} else if input.Type == "gmcInfo" {
		sqlStatement1 := `UPDATE dbo.cms_quote_item_master SET 
	gmsapprovalstatus=$1,
	gmc_rej_comments=$2 where quoteitemid=$3`

		_, err = db.Query(sqlStatement1,
			input.GmsApprovalStatus,
			input.GmsRejectionRemarks, input.LineItemId)
		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		return events.APIGatewayProxyResponse{200, headers, nil, string("success"), false}, nil
	} else if input.Type == "negativemarginapproval" {
		log.Println("Entered Negativemarging approval segment-MD")
		if input.Role=="Managing Director" || input.Role=="GMC" {
			log.Println("Role MD")
			sqlNMA1 :=`update dbo.cms_quote_item_master
						set
						negativemarginstatus=$1,
						negativemarginremarks=$2,
						negativemarginreason=$3
						where
						quoteitemid=$4`
			_, err = db.Query(sqlNMA1,
				input.NegativeMarginStatus, 
				NewNullString(input.NegativeMarginRemarks),
				NewNullString(input.NegativeMarginReason),
				input.LineItemId)
			if err != nil {
				log.Println(err)
				return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
			}
			return events.APIGatewayProxyResponse{200, headers, nil, string("success"), false}, nil
			//Send email
			
			e.Subject="Quote Line Item Negative Margin is "+input.NegativeMarginStatus 
			e.Message="Quote Line Item : "+strconv.Itoa(input.LineItemId)+".The negative margin is "+input.Margin+".It has been "+input.NegativeMarginStatus
			
			sqlStatementFUser1 := `SELECT emailid 
				FROM dbo.users_master_newpg where userid=$1`
			rows1, err1 := db.Query(sqlStatementFUser1,input.LoggedInUserID)
			if err1 != nil {
				log.Println(err1)
				return events.APIGatewayProxyResponse{500, headers, nil, err1.Error(), false}, nil
			}
			for rows1.Next() {
				err1 = rows1.Scan(&email)
				smtpSendEmail(QuoteTemp,e.Subject,e.Message,email)		
			}
	

		} else {
			return events.APIGatewayProxyResponse{500, headers, nil, string("Not authorised to perform negative margin approval"), false}, nil

		}


	} else if input.Type == "negativemarginrequest" {
		log.Println("Entered Negativemarging approval segment-ME")
		if input.Role=="Marketing Executive" || input.Role=="Managing Director" || input.Role=="GMC" {
			log.Println("Role : ME")
			sqlNMR1 :=`update dbo.cms_quote_item_master
						set
						negativemarginstatus=$1,
						margin=$2,
						marginpercentage=$3,
						finalprice=$4
						where
						quoteitemid=$5`
			_, err = db.Query(sqlNMR1,
				input.NegativeMarginStatus, 	
							input.Margin,
							input.MarginPercentage,
							input.FinalPrice,
							input.LineItemId)
			
			if err != nil {
				log.Println(err)
				return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
			}
			//Send email
			
			e.Subject="Quote Line Item Negative Margin Request"  
			e.Message="Quote Line Item : "+strconv.Itoa(input.LineItemId)+".The negative margin is "+input.Margin
			
			// sqlStatementFUser1 := `SELECT emailid 
			// 						FROM dbo.users_master_newpg 
			// 						where role='Managing Director' order by userid limit 1`
			// rows1, err1 := db.Query(sqlStatementFUser1)
			// if err1 != nil {
			// 	log.Println(err1)
			// 	return events.APIGatewayProxyResponse{500, headers, nil, err1.Error(), false}, nil
			// }
			// for rows1.Next() {
			// 	err1 = rows1.Scan(&email)
			// 	smtpSendEmail(QuoteTemp,e.Subject,e.Message,email)		
			// }

		}

	}

	// log.Println("Not updated the Quoteline item fields")
	return events.APIGatewayProxyResponse{200, headers, nil, string("Success"), false}, nil
}

func main() {
	lambda.Start(updateQuoteLineInfo)
}
func smtpSendEmail(temp,subject,message,to_email string) (string, error) {
	log.Println("Entered SMTP Email Module")
  // Receiver email address.
  to := []string{
    to_email,
  }
  // Authentication.
  auth := smtp.PlainAuth("", from_email, smtp_pass, smtpHost)

  t := template.Must(template.New(temp).Parse(temp))
 
  var body bytes.Buffer

  mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
  body.Write([]byte(fmt.Sprintf("Subject:"+subject+"\n%s\n\n",mimeHeaders)))
//   body.Write([]byte(fmt.Sprintf("Subject: This is a test subject \n%s\n\n", mimeHeaders)))

  t.Execute(&body, struct {
    
    EMessage string
	
  }{
    
    EMessage: message,
	
  })

  // Sending email.
  err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from_email, to, body.Bytes())
  if err != nil {
    fmt.Println(err)
    
  }
  return "Email Sent!",nil
}
