export async function request(method: string, path: string, body?: any){
  const r = await fetch(path, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: body ? JSON.stringify(body) : undefined,
  })
  const text = await r.text()
  try {
    return JSON.parse(text)
  } catch {
    return text
  }
}
export const get = (p: string) => request('GET', p)
export const post = (p: string, b?: any) => request('POST', p, b)
export const put = (p: string, b?: any) => request('PUT', p, b)
export const del = (p: string) => request('DELETE', p)
