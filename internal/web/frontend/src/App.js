import React, { useEffect, useState } from 'react';

function App() {
  const [fetchedData, setFetchedData] = useState(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await fetch('/api/v1/features');

        if (response.ok) {
          const data = await response.json();
          console.log('Response from /api/v1/features:', data);
          setFetchedData(data); // Set the fetched data in state
        } else {
          console.error('Failed to fetch data:', response.statusText);
        }
      } catch (error) {
        console.error('Error fetching data:', error);
      }
    };

    fetchData();
  }, []);

  return (
    <div className="App" style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
      {fetchedData ? (
        <div style={{ textAlign: 'center' }}>
          <strong>Fetched Data:</strong>
          <p>{JSON.stringify(fetchedData)}</p>
        </div>
      ) : (
        <p>Loading...</p>
      )}
    </div>
  );
}

export default App;
