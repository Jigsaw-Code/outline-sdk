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
    # Calculate median durations for A and HTTPS queries for each domain
    median_durations = df.groupby(['domain', 'query_type'])['duration_ms'].median().unstack()
    
    # Ensure 'A' and 'HTTPS' columns exist
    if 'A' not in median_durations.columns or 'HTTPS' not in median_durations.columns:
        print("Data for 'A' or 'HTTPS' queries not found.")
        return

    # Calculate duration difference
    median_durations['duration_diff'] = median_durations['HTTPS'] - median_durations['A']

    # Filter for domains where median HTTPS is > 50ms slower than median A
    slow_domains = median_durations[median_durations['duration_diff'] > 50].index

    if slow_domains.empty:
        print("No domains found where median HTTPS query is > 50ms slower than median A query.")
        return

    # --- 2. Get all runs for slow domains ---
    slow_df = df[df['domain'].isin(slow_domains)]

    # --- 3. Pivot data to have runs as columns ---
    # Get A query durations for slow domains
    a_durations = slow_df[slow_df['query_type'] == 'A'][['domain', 'run', 'duration_ms']].set_index(['domain', 'run'])
    
    # Get HTTPS query durations and pivot
    https_pivot = slow_df[slow_df['query_type'] == 'HTTPS'].pivot_table(
        index='domain', columns='run', values='duration_ms'
    )

    # --- 4. Generate Markdown Table ---
    markdown_table = "| Domain | Rank | Run 1 | Run 2 | Run 3 | Run 4 | Run 5 |\n"
    markdown_table += "|:---|:---|:---|:---|:---|:---|:---|"

    domain_ranks = df[['domain', 'rank']].drop_duplicates().set_index('domain')

    for domain, row in https_pivot.iterrows():
        rank = domain_ranks.loc[domain]['rank']
        markdown_table += f"\n| {domain} | {rank:.0f} |"
        for run in range(1, 6):
            https_duration = row.get(run)
            
            a_duration = None
            try:
                a_duration = a_durations.loc[(domain, run)]['duration_ms']
            except KeyError:
                pass # No matching A query for this run

            if https_duration is not None and a_duration is not None:
                diff = https_duration - a_duration
                cell = f"{https_duration:.0f} ({diff:+.0f})"
                if diff > 50:
                    cell = f"**{cell}**"
                markdown_table += f" {cell} |"
            else:
                markdown_table += " - |"
    
    print(markdown_table)

if __name__ == '__main__':
    main()

