import NotebookCell from './NotebookCell';

export default function NotebookCanvas() {
  return (
    <div className="max-w-4xl mx-auto py-12 px-8 space-y-8">
      <NotebookCell 
        type="markdown" 
        content="## Initial Data Exploration
Let's load the Q3 earnings data and correlate it with the macro environment." 
      />
      
      <NotebookCell 
        type="code" 
        content="import pandas as pd

df = pd.read_csv('Q3_Earnings.csv')
df.describe()" 
        output={{ type: 'text', data: '       Revenue        EPS\ncount  12000.0   12000.0\nmean    450.2      2.1' }}
      />
      
      <NotebookCell 
        type="ai" 
        content="Generate a scatter plot of Revenue vs EPS, color coded by sector, and add a regression line." 
        output={{ type: 'chart', data: [
          { name: 'Jan', value: 400 },
          { name: 'Feb', value: 300 },
          { name: 'Mar', value: 600 },
          { name: 'Apr', value: 800 },
          { name: 'May', value: 700 }
        ] }}
      />

      <div className="pt-8 pb-32 flex justify-center opacity-0 hover:opacity-100 transition-opacity">
        <button className="flex items-center gap-2 px-4 py-2 rounded-full bg-white/5 border border-white/10 text-text-muted text-sm hover:text-white hover:bg-white/10 transition-colors">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" /></svg>
          Add Block
        </button>
      </div>
    </div>
  );
}
