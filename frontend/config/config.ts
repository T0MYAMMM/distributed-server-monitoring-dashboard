const getApiUrl = () => {
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL;
  }
  
  if (typeof window !== 'undefined') {
    const protocol = window.location.protocol;
    const hostname = window.location.hostname;
    return `${protocol}//${hostname}:5000`;
  }
  
  return 'http://localhost:5000';
};

export const API_URL = getApiUrl();

// Derive the dashboard WebSocket URL from the API base, swapping the scheme.
export const WS_URL = API_URL.replace(/^http/, 'ws') + '/api/ws/dashboard';
