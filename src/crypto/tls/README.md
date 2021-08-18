# TLS

TLS: Transport Layer Security Protocol

在TCP协议之上。

## 分类

1. TLS Record Protocol
2. TLS Handshake Protocol

记录协议提供安全保证有2个基本属性：
1. 连接是私有的。
2. 连接时可靠的。

握手协议允许客户端和服务器验证彼此，协议加密算法和加密key。
其有3个基本属性：
1. 两端的身份可以被验证通过不对称加解密
2. 共享秘密的谈判是安全的
3. 协商是可靠的：没有攻击者不能在没有被通信双方检测到的情况下修改协商通信。