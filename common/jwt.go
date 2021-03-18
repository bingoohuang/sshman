package common

import (
	"github.com/bingoohuang/sshman/config"
	"github.com/dgrijalva/jwt-go"
	"time"
)

type Claims struct {
	Userid uint
	jwt.StandardClaims
}

func ReleaseToken(id uint) (token string, err error) {
	expireTime := time.Now().Add(3 * 24 * time.Hour)
	claims := &Claims{
		Userid: id,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "admin",
			Subject:   "user",
		},
	}
	token_obj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err = token_obj.SignedString([]byte(config.Conf.Jwt.Key))
	return
}

func ParseToken(token string) (*Claims, error) {
	//用于解析鉴权的声明，方法内部主要是具体的解码和校验的过程，最终返回*Token
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(_ *jwt.Token) (interface{}, error) {
		return []byte(config.Conf.Jwt.Key), nil
	})

	if tokenClaims != nil {
		// 从tokenClaims中获取到Claims对象，并使用断言，将该对象转换为我们自己定义的Claims
		// 要传入指针，项目中结构体都是用指针传递，节省空间。
		if claims, ok := tokenClaims.Claims.(*Claims); ok && tokenClaims.Valid {
			return claims, nil
		}
	}
	return nil, err

}
