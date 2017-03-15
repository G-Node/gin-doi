package ginDoi

import "testing"

func TestGinOAUTH(t *testing.T) {
	pr := OauthProvider{Uri: "https://auth.gin.g-node.org/api/accounts"}
	pr.getUser("testi", "Bearer 6NV23M4XALWCZKTB2TRR67R7E7WU66BHYS7J45Q3ZKSP3LTWESXVIEWPORIET32X4PAIJQBYKT7ONNEMVMWCNUHYITAWLL576OGNZMY")
	t.Fail()
}
