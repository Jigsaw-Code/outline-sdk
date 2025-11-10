import pandas as pd
import json
import os

def main():
    report_dir = '/Users/fortuna/firehook/outline-sdk/x/tools/ech-test/report'
    csv_file = os.path.join(report_dir, 'results-top1000-n5.csv')

    try:
        df = pd.read_csv(csv_file)
    except FileNotFoundError:
        print(f"Error: File not found at {csv_file}")
        exit()

    # Filter out rows where query_type is not A or HTTPS
    filtered_df = df[df['query_type'].isin(['A', 'HTTPS'])].copy()

    # Extract rank for each domain
    domain_ranks = df[['domain', 'rank']].drop_duplicates().set_index('domain')

    # Convert 'answers' column from string to list/json object
    # This is necessary to correctly identify successful queries
    def parse_answers(answers_str):
        try:
            parsed = json.loads(answers_str)
            return parsed if parsed else []
        except json.JSONDecodeError:
            return []

    filtered_df['parsed_answers'] = filtered_df['answers'].apply(parse_answers)

    # Define a function to check if a query has a valid answer
    def has_valid_answer(row):
        if row['query_type'] == 'HTTPS':
            # For HTTPS, check if there are any answers and if rcode is NOERROR
            return len(row['parsed_answers']) > 0 and row['rcode'] == 'NOERROR'
        elif row['query_type'] == 'A':
            # For A queries, check if there are any answers and if rcode is NOERROR
            return len(row['parsed_answers']) > 0 and row['rcode'] == 'NOERROR'
        return False

    filtered_df['has_answer'] = filtered_df.apply(has_valid_answer, axis=1)

    # Calculate median, min, and max durations for A and HTTPS queries for each domain
    grouped_durations = filtered_df.groupby(['domain', 'query_type'])['duration_ms']
    median_durations = grouped_durations.median().unstack()
    min_durations = grouped_durations.min().unstack()
    max_durations = grouped_durations.max().unstack()

    # Merge A and HTTPS durations (median, min, max)
    merged_durations = pd.DataFrame({
        'median_A_duration_ms': median_durations['A'],
        'median_HTTPS_duration_ms': median_durations['HTTPS'],
        'min_HTTPS_duration_ms': min_durations['HTTPS'],
        'max_HTTPS_duration_ms': max_durations['HTTPS'],
    }).reset_index()

    # Add rank to merged_durations
    merged_durations = merged_durations.set_index('domain').join(domain_ranks).reset_index()

    # Calculate duration difference and ratio
    merged_durations['duration_diff'] = merged_durations['median_HTTPS_duration_ms'] - merged_durations['median_A_duration_ms']
    merged_durations['ratio'] = merged_durations['median_HTTPS_duration_ms'] / merged_durations['median_A_duration_ms']

    # Filter domains where duration_diff > 50ms
    filtered_domains = merged_durations[merged_durations['duration_diff'] > 50].copy()

    # Sort by min_HTTPS_duration_ms in descending order
    filtered_domains.sort_values(by='min_HTTPS_duration_ms', ascending=False, inplace=True)

    # Generate Markdown table
    markdown_table = "| Domain | Rank | Median A (ms) | Min HTTPS (ms) | Median HTTPS (ms) | Max HTTPS (ms) | Ratio (HTTPS/A) |\n"
    markdown_table += "|:---|:---|:---|:---|:---|:---|:---|"

    for index, row in filtered_domains.iterrows():
        markdown_table += f"\n| {row['domain']} | {row['rank']:.0f} | {row['median_A_duration_ms']:.0f} | {row['min_HTTPS_duration_ms']:.0f} | {row['median_HTTPS_duration_ms']:.0f} | {row['max_HTTPS_duration_ms']:.0f} | {row['ratio']:.2f} |"

    print(markdown_table)

if __name__ == '__main__':
    main()
