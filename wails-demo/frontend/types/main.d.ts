import { LitElement } from "lit";
export declare class MainPage extends LitElement {
    static localization: {
        getLocale: (() => string) & {
            _LIT_LOCALIZE_GET_LOCALE_?: undefined;
        };
        setLocale: ((newLocale: string) => Promise<void>) & {
            _LIT_LOCALIZE_SET_LOCALE_?: undefined;
        };
    };
    static styles: import("lit").CSSResult;
    get formData(): {
        accessKey: string;
        domain: string;
        resolvers: string[];
        protocols: {
            tcp: boolean;
            udp: boolean;
        };
    } | null;
    get locale(): string;
    set locale(newLocale: string);
    render(): import("lit-html").TemplateResult<1>;
    testConnectivity(event: SubmitEvent): Promise<void>;
    updateLocale(event: Event): void;
}
declare global {
    interface HTMLElementTagNameMap {
        "main-page": MainPage;
    }
}
