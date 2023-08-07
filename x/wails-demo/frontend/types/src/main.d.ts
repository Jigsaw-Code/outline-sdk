import { LitElement } from "lit";
import { main } from "../wailsjs/go/models";
export declare class MainPage extends LitElement {
    get locale(): string;
    set locale(newLocale: string);
    get formData(): {
        accessKey: string;
        domain: string;
        resolvers: string[];
        protocols: {
            tcp: boolean;
            udp: boolean;
        };
    } | null;
    testResults: main.ConnectivityTestResult[] | Error | null;
    testConnectivity(event: SubmitEvent): Promise<void>;
    static styles: import("lit").CSSResult;
    render(): import("lit-html").TemplateResult<1>;
    renderResults(): import("lit-html").TemplateResult<1> | undefined;
    renderResultsList(results: main.ConnectivityTestResult[] | Error): import("lit-html").TemplateResult<1>;
}
declare global {
    interface HTMLElementTagNameMap {
        "main-page": MainPage;
    }
}
