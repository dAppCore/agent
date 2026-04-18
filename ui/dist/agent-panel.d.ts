import { LitElement } from 'lit';
/**
 * Agent dashboard panel — shows issues, sprint progress, and fleet status.
 * Works in core/ide (Wails), lthn.sh (Laravel), and standalone browsers.
 *
 * @element core-agent-panel
 */
export declare class CoreAgentPanel extends LitElement {
    apiUrl: string;
    apiKey: string;
    private issues;
    private sprint;
    private loading;
    private error;
    private activeTab;
    static styles: import("lit").CSSResult;
    connectedCallback(): void;
    private fetchData;
    private setTab;
    private renderIssues;
    private renderSprint;
    render(): import("lit-html").TemplateResult<1>;
}
