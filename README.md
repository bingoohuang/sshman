# sshman

![](https://img.shields.io/github/license/bingoohuang/sshman)
![](https://img.shields.io/github/stars/bingoohuang/sshman)
![](https://img.shields.io/github/forks/bingoohuang/sshman)
![](https://img.shields.io/github/issues/bingoohuang/sshman)

go版本多用户webssh管理工具

服务端不保存用户明文密码，且不保存解密秘钥，如需对其他用户开放，请不要修改此部分代码，以免造成不必要的损失！

## 开发框架

- Gin + gorm
- Lauyi + Xterm.js

## 更新日志

    2020/12/17 新增WEB_SFTP功能，拖动文件到终端窗口里即可上传
    2020/12/16 前端新增文件/文件夹拖动到Terminal的自动解析功能（SFTP需要），修改layer弹出窗口逻辑，增加回车提交事件
    2020/12/14 修复无操作自动断开、修复网络延迟造成的js加载延迟问题

## 开发计划

- [-] ssh功能
- [-] sftp文件上传功能

## 环境

> Mysql
> Redis

## 配置文件

修改config.toml的相关参数，短信接口使用阿里云短信

## 补充说明

如需要使用Nginx等进行反代，请确保可以正常代理websocket

