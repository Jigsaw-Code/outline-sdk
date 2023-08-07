export declare namespace main {
    class ConnectivityTestError {
        operation: string;
        posixError: string;
        message: string;
        static createFrom(source?: any): ConnectivityTestError;
        constructor(source?: any);
    }
    class ConnectivityTestProtocolConfig {
        tcp: boolean;
        udp: boolean;
        static createFrom(source?: any): ConnectivityTestProtocolConfig;
        constructor(source?: any);
    }
    class ConnectivityTestResult {
        proxy: string;
        resolver: string;
        proto: string;
        prefix: string;
        time: any;
        durationMs: number;
        error?: ConnectivityTestError;
        static createFrom(source?: any): ConnectivityTestResult;
        constructor(source?: any);
        convertValues(a: any, classs: any, asMap?: boolean): any;
    }
}
