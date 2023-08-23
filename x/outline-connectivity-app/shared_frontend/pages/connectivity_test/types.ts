export interface ConnectivityTestPageSubmitInput {
  accessKey: string;
  domain: string;
  resolvers: string[];
  protocols: {
    tcp: boolean;
    udp: boolean;
  };
}

export interface ConnectivityTestPageSubmitResult {
  time: string;
  durationMs: number;
  proto: string;
  resolver: string;
  error?: {
    operation: string;
    posixError: string;
    message: string;
  };
}
