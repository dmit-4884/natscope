interface NavigatorUAData {
  platform?: string
}

export function isMacPlatform(): boolean {
  if (typeof navigator === 'undefined') return false

  const uaData = (navigator as Navigator & { userAgentData?: NavigatorUAData }).userAgentData
  if (uaData?.platform) {
    return uaData.platform.toLowerCase().includes('mac')
  }

  return /mac/i.test(navigator.userAgent)
}
