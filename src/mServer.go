package ginDoi

import (
	"net/smtp"
	"log"
)

var (
	MAILLOG = "MailServer"
)

type MailServer struct {
	Adress string
	From string
	DoSend bool
	Master string
}

func (ms *MailServer) ToMaster(content string) error {
	auth := smtp.PlainAuth("", "", "", ms.Adress)
	if ms.DoSend {
		return smtp.SendMail(ms.Adress, auth, ms.From, []string{ms.Master}, []byte(content))
	}
	log.Printf("[%s] Fake Mail to: %s, content: %s, Auth:%+v", MAILLOG, ms.Master, content, auth)
	return nil
}