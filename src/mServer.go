package ginDoi

import (
	log "github.com/Sirupsen/logrus"
	"net/smtp"
)

var (
	MAILLOG = "MailServer"
)

type MailServer struct {
	Adress string
	From   string
	DoSend bool
	Master string
}

func (ms *MailServer) ToMaster(content string) error {
	auth := smtp.PlainAuth("", "", "", ms.Adress)
	if ms.DoSend {
		return smtp.SendMail(ms.Adress, auth, ms.From, []string{ms.Master}, []byte(content))
	}
	log.WithFields(log.Fields{
		"source": MAILLOG,
	}).Infof("Fake Mail to: %s, content: %s, Auth:%+v", ms.Master, content, auth)
	return nil
}
