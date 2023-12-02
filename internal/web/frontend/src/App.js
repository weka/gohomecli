import React, { useEffect } from 'react';
import logo from './logo.svg';

function App() {
  useEffect(() => {
    const fetchData = async () => {
      try {
        // Fetch data from the /api/calculate endpoint on the same host and port
        const response = await fetch('/api/calculate');

        // Check if the request was successful (status code 200)
        if (response.ok) {
          const data = await response.json();
          console.log('Response from /api/calculate:', data);
        } else {
          console.error('Failed to fetch data:', response.statusText);
        }
      } catch (error) {
        console.error('Error fetching data:', error);
      }
    };

    // Call the fetchData function
    fetchData();
  }, []); // Empty dependency array to run the effect only once on component mount

  return (
    <div className="App">
      <header className="App-header">
        <img src={logo} className="App-logo" alt="logo" />
        <p>
          Edit <code>src/App.js</code> and save to reload.
        </p>
        <a
          className="App-link"
          href="https://reactjs.org"
          target="_blank"
          rel="noopener noreferrer"
        >
          Learn React
        </a>
      </header>
    </div>
  );
}

export default App;
