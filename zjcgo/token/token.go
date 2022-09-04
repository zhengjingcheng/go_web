package token

import (
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/zhengjingcheng/zjcgo"
	"net/http"
	"time"
)

const JWTToken = "zjcgo_token"

type JwtHandler struct {
	//jwt算法
	Alg string
	//过期时间
	TimeOut time.Duration
	//时间函数  从此时开始计算过期
	RefreshTimeOut time.Duration

	TimeFun func() time.Time
	//登录认证方法
	Authenticator func(ctx *zjcgo.Context) (map[string]any, error)
	//私钥
	Privatekey string
	//key
	Key []byte
	//刷新Key
	RefreshKey string
	//save cookie
	SendCookie     bool
	CookieName     string
	CookieMaxAge   int64
	CookieDomain   string
	SecureCookie   bool
	CookieHTTPOnly bool
	Header         string
	AuthHandler    func(ctx *zjcgo.Context, err error)
}

//登录 用户认证（用户名密码）-> 用户id 将id生成jwt，并保存到cookie或者进行返回

type JwtResponse struct {
	Token        string
	RefreshToken string //刷新token防止token过期
}

//登录
func (j *JwtHandler) LoginHandler(ctx *zjcgo.Context) (*JwtResponse, error) {
	//先进行登录验证 拿到用户名和密码
	data, err := j.Authenticator(ctx)
	if err != nil {
		//如果报错
		return nil, err
	}
	//实现A,B,C三部分
	if j.Alg == "" {
		//如果不设置就给一个默认值
		j.Alg = "HS256"
	}
	//A部分
	//检索出签名方法
	signingMethod := jwt.GetSigningMethod(j.Alg)
	//利用签名方法创建出一个新的token
	token := jwt.New(signingMethod)
	//把数据放进去
	//B部分
	claims := token.Claims.(jwt.MapClaims)
	if data != nil {
		for k, v := range data {
			claims[k] = v
		}
	}
	//如果没设置时间就返回当前时间吗，】
	if j.TimeFun == nil {
		j.TimeFun = func() time.Time {
			return time.Now()
		}
	}
	expire := j.TimeFun().Add(j.TimeOut)
	//还需要自己额外设置一部分数据
	//过期时间
	claims["exp"] = expire.Unix()
	//发布时间当前时间
	claims["iat"] = j.TimeFun().Unix() //以秒为单位返回
	//c部分，生成token
	var tokenString string
	var errToken error
	if j.usingPublicKeyAlgo() {
		//如果这个算法需要使用私钥
		tokenString, errToken = token.SignedString(j.Privatekey)
	} else {
		//否则直接使用key
		tokenString, errToken = token.SignedString(j.Key)
	}
	if errToken != nil {
		return nil, errToken
	}
	//生成token
	jr := &JwtResponse{
		Token:        tokenString,
		RefreshToken: tokenString,
	}
	//发送存储cookie
	if j.SendCookie {
		if j.CookieName == "" {
			//默认值
			j.CookieName = JWTToken
		}
		if j.CookieMaxAge == 0 {
			j.CookieMaxAge = expire.Unix() - j.TimeFun().Unix()
		}
		ctx.SetCookie(j.CookieName, tokenString, int(j.CookieMaxAge), "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
	}
	//refreshToken类似token生成
	refreshToken, err := j.refreshToken(token)
	if err != nil {
		return nil, errToken
	}
	jr.RefreshToken = refreshToken
	return jr, nil
}

//判断方法
func (j *JwtHandler) usingPublicKeyAlgo() bool {
	switch j.Alg {
	case "RS256", "RS512", "RS384":
		return true
	}
	return false
}

func (j *JwtHandler) refreshToken(token *jwt.Token) (string, error) {
	claims := token.Claims.(jwt.MapClaims)
	//修改过期时间
	claims["exp"] = j.TimeFun().Add(j.RefreshTimeOut).Unix()
	var tokenString string
	var errToken error
	if j.usingPublicKeyAlgo() {
		//如果这个算法需要使用私钥
		tokenString, errToken = token.SignedString(j.Privatekey)
	} else {
		//否则直接使用key
		tokenString, errToken = token.SignedString(j.Key)
	}
	if errToken != nil {
		return "", errToken
	}
	return tokenString, nil
}

//退出登录
func (j *JwtHandler) LogoutHandler(ctx *zjcgo.Context) error {
	//如果有cookie就将cookie删掉
	if j.SendCookie {
		if j.CookieName == "" {
			j.CookieName = JWTToken
		}
		ctx.SetCookie(j.CookieName, "", -1, "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
		return nil
	}
	return nil
}

//刷新token(不需要走登录逻辑，只需要通过上下文中拿到刷新的key)
func (j *JwtHandler) RefreshHandler(ctx *zjcgo.Context) (*JwtResponse, error) {
	rToken, ok := ctx.Get(j.RefreshKey)
	if !ok {
		//如果报错
		return nil, errors.New("refresh token is null")
	}
	//实现A,B,C三部分
	if j.Alg == "" {
		//如果不设置就给一个默认值
		j.Alg = "HS256"
	}
	//解析token
	t, err := jwt.Parse(rToken.(string), func(token *jwt.Token) (interface{}, error) {
		if j.usingPublicKeyAlgo() {
			//如果这个算法需要使用私钥
			return j.Privatekey, nil
		} else {
			//否则直接使用key
			return j.Key, nil
		}
	})
	if err != nil {
		//如果refreshtoken不合法
		return nil, err
	}
	claims := t.Claims.(jwt.MapClaims)

	//如果没设置时间就返回当前时间吗，】
	if j.TimeFun == nil {
		j.TimeFun = func() time.Time {
			return time.Now()
		}
	}
	expire := j.TimeFun().Add(j.TimeOut)
	//还需要自己额外设置一部分数据
	//过期时间
	claims["exp"] = expire.Unix()
	//发布时间当前时间
	claims["iat"] = j.TimeFun().Unix() //以秒为单位返回
	//c部分，生成token
	var tokenString string
	var errToken error
	if j.usingPublicKeyAlgo() {
		//如果这个算法需要使用私钥
		tokenString, errToken = t.SignedString(j.Privatekey)
	} else {
		//否则直接使用key
		tokenString, errToken = t.SignedString(j.Key)
	}
	if errToken != nil {
		return nil, errToken
	}
	//生成token
	jr := &JwtResponse{
		Token:        tokenString,
		RefreshToken: tokenString,
	}
	//发送存储cookie
	if j.SendCookie {
		if j.CookieName == "" {
			//默认值
			j.CookieName = JWTToken
		}
		if j.CookieMaxAge == 0 {
			j.CookieMaxAge = expire.Unix() - j.TimeFun().Unix()
		}
		ctx.SetCookie(j.CookieName, tokenString, int(j.CookieMaxAge), "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
	}
	//refreshToken类似token生成
	refreshToken, err := j.refreshToken(t)
	if err != nil {
		return nil, errToken
	}
	jr.RefreshToken = refreshToken
	return jr, nil
}

//jwt 登录中间件
//判断header 的token是否合法

func (j *JwtHandler) AuthInterceptor(next zjcgo.HandlerFunc) zjcgo.HandlerFunc {
	return func(ctx *zjcgo.Context) {
		//
		if j.Header == "" {
			j.Header = "Authorization"
		}
		token := ctx.R.Header.Get(j.Header)
		if token == "" {
			//从缓冲中获得
			if j.SendCookie {
				token = ctx.GetCookie(j.CookieName)
				if token == "" {
					if j.AuthHandler == nil {
						ctx.W.WriteHeader(http.StatusUnauthorized)
					} else {
						j.AuthHandler(ctx, nil)
					}
					return
				}
			}
		}
		//解析token
		t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			if j.usingPublicKeyAlgo() {
				return []byte(j.Privatekey), nil
			}
			return []byte(j.Key), nil
		})
		if err != nil {
			if j.AuthHandler == nil {
				ctx.W.WriteHeader(http.StatusUnauthorized)
			} else {
				j.AuthHandler(ctx, err)
			}
			return
		}
		claims := t.Claims.(jwt.MapClaims)
		ctx.Set("claims", claims)
		next(ctx)
	}
}
