// Access Token 内存持有器：按设计文档 5.3 节，Token 只存内存，
// 不写入 localStorage/sessionStorage，降低 XSS 窃取风险。
let accessToken: string | null = null

export function setAccessToken(token: string | null) {
  accessToken = token
}

export function getAccessToken(): string | null {
  return accessToken
}

export function clearAccessToken() {
  accessToken = null
}
