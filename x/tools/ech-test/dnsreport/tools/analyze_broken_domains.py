import pandas as pd
import sys
import subprocess

def run_dig_command(command):
    try:
        result = subprocess.run(command, capture_output=True, text=True, check=True)
        return result.stdout.strip()
    except subprocess.CalledProcessError as e:
        return f"Error executing command: {e}\n{e.stderr.strip()}"
    except FileNotFoundError:
        return "Error: 'dig' command not found. Please ensure 'dig' is installed and in your PATH."

def analyze_broken_domains_python(input_file):
    try:
        df = pd.read_csv(input_file)
    except FileNotFoundError:
        print(f"Error: File not found at {input_file}")
        sys.exit(1)

    https_df = df[df['query_type'] == 'HTTPS']

    broken_domains_info = {}
    for domain, group in https_df.groupby('domain'):
        if (group['duration_ms'] > 2000).all():
            errors = group['error'].dropna().unique().tolist()
            rcodes = group['rcode'].dropna().unique().tolist()
            broken_domains_info[domain] = {'errors': errors, 'rcodes': rcodes, 'dig_analysis': {}}

            # Get authoritative nameservers
            ns_output = run_dig_command(['dig', '+short', 'NS', domain])
            broken_domains_info[domain]['dig_analysis']['ns_servers_raw'] = ns_output
            
            ns_servers = [ns.strip() for ns in ns_output.split('\n') if ns.strip()]

            if ns_servers:
                broken_domains_info[domain]['dig_analysis']['authoritative_queries'] = {}
                for ns in ns_servers:
                    https_query_output = run_dig_command(['dig', f'@{ns}', domain, 'HTTPS'])
                    broken_domains_info[domain]['dig_analysis']['authoritative_queries'][ns] = https_query_output
            else:
                broken_domains_info[domain]['dig_analysis']['authoritative_queries'] = "No authoritative nameservers found."

    return broken_domains_info

def generate_markdown_report(broken_domains_analysis, output_file):
    with open(output_file, 'w') as f:
        f.write("# Broken Domains Analysis Report\n\n")
        f.write("This report details domains where HTTPS queries consistently timed out (> 2s).\n\n")

        if not broken_domains_analysis:
            f.write("No broken domains found.\n")
            return

        for domain, info in broken_domains_analysis.items():
            f.write(f"## Domain: {domain}\n\n")
            f.write("### Initial Analysis (from CSV data)\n\n")
            f.write(f"- **RCODEs:** {', '.join(info['rcodes']) if info['rcodes'] else 'N/A'}\n")
            f.write(f"- **Errors:** {', '.join(info['errors']) if info['errors'] else 'N/A'}\n\n")

            f.write("### Authoritative DNS Investigation\n\n")
            f.write("#### Authoritative Nameservers (dig +short NS)\n")
            f.write("```\n")
            f.write(info['dig_analysis']['ns_servers_raw'] + "\n")
            f.write("```\n\n")

            if isinstance(info['dig_analysis']['authoritative_queries'], dict):
                for ns, query_output in info['dig_analysis']['authoritative_queries'].items():
                    f.write(f"#### Querying {ns} for HTTPS record\n")
                    f.write("```\n")
                    f.write(query_output + "\n")
                    f.write("```\n\n")
            else:
                f.write(f"**{info['dig_analysis']['authoritative_queries']}**\n\n")

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: python analyze_broken_domains.py <input_csv_file>")
        sys.exit(1)

    input_file = sys.argv[1]
    output_report_file = "broken_domains_report.md"

    print(f"Analyzing broken domains from {input_file}...")
    broken_domains_analysis = analyze_broken_domains_python(input_file)
    generate_markdown_report(broken_domains_analysis, output_report_file)
    print(f"Report generated: {output_report_file}")
