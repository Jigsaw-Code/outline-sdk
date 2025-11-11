import pandas as pd
import json
import sys

def main():
    if len(sys.argv) < 2:
        print("Usage: python generate_filtered_table.py <input_csv_file>")
        sys.exit(1)

    input_file = sys.argv[1]

    try:
        df = pd.read_csv(input_file)
    except FileNotFoundError:
        print(f"Error: File not found at {input_file}")
        sys.exit(1)

    # --- 1. Find slow domains ---
    median_durations = df.groupby(['domain', 'query_type'])['duration_ms'].median().unstack()
    
    if 'A' not in median_durations.columns or 'HTTPS' not in median_durations.columns:
        print("Data for 'A' or 'HTTPS' queries not found.")
        return

    median_durations['duration_diff'] = median_durations['HTTPS'] - median_durations['A']
    slow_domains_series = median_durations[median_durations['duration_diff'] > 50].index

    if slow_domains_series.empty:
        print("No domains found where median HTTPS query is > 50ms slower than median A query.")
        return

    # --- 2. Get all runs for slow domains ---
    slow_df = df[df['domain'].isin(slow_domains_series)]

    # --- New Step: Sort slow_domains by Min HTTPS duration (slowest first) ---
    min_https_durations = {}
    for domain in slow_domains_series:
        https_durations = slow_df[(slow_df['domain'] == domain) & (slow_df['query_type'] == 'HTTPS')]['duration_ms'].tolist()
        if https_durations:
            min_https_durations[domain] = min(https_durations)
        else:
            min_https_durations[domain] = 0 # Default to 0 if no HTTPS data

    # Sort domains by their minimum HTTPS duration in descending order
    sorted_slow_domains = sorted(slow_domains_series, key=lambda d: min_https_durations.get(d, 0), reverse=True)

    # --- 3. Process each domain ---
    markdown_table = "| Domain | Rank | Min | 2nd | Median | 4th | Max |\n"
    markdown_table += "|:---|:---|:---|:---|:---|:---|:---|"

    domain_ranks = df[['domain', 'rank']].drop_duplicates().set_index('domain')

    for domain in sorted_slow_domains: # Use the sorted list of domains
        rank = domain_ranks.loc[domain]['rank']
        
        domain_runs = []
        for run in range(1, 6):
            https_run_df = slow_df[(slow_df['domain'] == domain) & (slow_df['query_type'] == 'HTTPS') & (slow_df['run'] == run)]
            a_run_df = slow_df[(slow_df['domain'] == domain) & (slow_df['query_type'] == 'A') & (slow_df['run'] == run)]

            if not https_run_df.empty and not a_run_df.empty:
                https_duration = https_run_df['duration_ms'].iloc[0]
                a_duration = a_run_df['duration_ms'].iloc[0]
                diff = https_duration - a_duration
                
                cell = f"{https_duration:.0f} ({diff:+.0f})"
                if diff > 50:
                    cell = f"**{cell}**"
                
                domain_runs.append({'duration': https_duration, 'cell': cell})

        # Sort runs by HTTPS duration (already done in previous step, but this is for the cells within the row)
        domain_runs.sort(key=lambda x: x['duration'])

        markdown_table += f"\n| {domain} | {rank:.0f} |"
        for i in range(5):
            if i < len(domain_runs):
                markdown_table += f" {domain_runs[i]['cell']} |"
            else:
                markdown_table += " - |"
    
    print(markdown_table)

if __name__ == '__main__':
    main()


