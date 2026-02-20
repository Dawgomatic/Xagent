/**
 * Zillow data fetcher
 * Uses RapidAPI's unofficial Zillow scraper API (most accessible free option)
 *
 * FREE OPTIONS:
 *   1. RapidAPI "Zillow-com1" or "Walk Score" - 30 free req/month
 *   3. Direct scraping with ScraperAPI (1000 free credits)
 *
 * SIGNUP REQUIRED:
 *   - RapidAPI: https://rapidapi.com (free, credit card optional for basic tier)
 */

const axios = require('axios');

const RAPIDAPI_KEY = process.env.RAPIDAPI_KEY;

// RapidAPI Zillow endpoint - US Property Market (600 free/mo, fastest)
const RAPIDAPI_ZILLOW_HOST = 'us-property-market1.p.rapidapi.com';

/**
 * Fetch Zillow for-sale listings by ZIP code
 * @param {string} zip - ZIP code or city string
 * @param {object} options
 * @returns {Array} Zillow listing objects
 */
async function fetchZillowListings(zip, options = {}) {
  const { maxResults = 40, minPrice, maxPrice, beds, propertyType } = options;

      'Get free key at: https://rapidapi.com');
  }

  if (RAPIDAPI_KEY) {
    return fetchViaRapidAPI(zip, options);
  }

  }

  throw new Error('No API keys configured. Set RAPIDAPI_KEY in .env\nGet free key at: https://rapidapi.com');
}

/**
 * RapidAPI Zillow scraper (30 free req/month)
 */
async function fetchViaRapidAPI(zip, options = {}) {
  const { maxResults = 40 } = options;

  process.stderr.write(`  📡 Fetching Zillow listings for ${zip} via RapidAPI...\n`);

  try {
    // US Property Market API - search by location
    const searchResponse = await axios.get(`https://${RAPIDAPI_ZILLOW_HOST}/search`, {
      params: {
        location: zip,
        status: 'forSale'
      },
      headers: {
        'X-RapidAPI-Key': RAPIDAPI_KEY,
        'X-RapidAPI-Host': RAPIDAPI_ZILLOW_HOST
      },
      timeout: 15000
    });

    const results = searchResponse.data?.props || searchResponse.data?.results || searchResponse.data?.data || [];
    const resultArray = Array.isArray(results) ? results : (searchResponse.data?.props ? Object.values(searchResponse.data.props) : []);
    process.stderr.write(`  ✅ Found ${resultArray.length} Zillow listings\n`);

    return resultArray.slice(0, maxResults).map(normalizeZillowListing);

  } catch (err) {
    if (err.response?.status === 429) {
      throw new Error('RapidAPI rate limit hit. Free tier = 600 requests/month.\n' +
        'Upgrade at: https://rapidapi.com/SwongF/api/us-property-market1');
    }
    if (err.response?.status === 403 || err.response?.status === 401) {
      throw new Error('RapidAPI auth failed. Subscribe to US Property Market API (free):\n' +
        'https://rapidapi.com/SwongF/api/us-property-market1');
    }
    // Log response for debugging
    if (err.response?.data) {
      process.stderr.write(`  ⚠️ API response: ${JSON.stringify(err.response.data).slice(0, 200)}\n`);
    }
    throw new Error(`Zillow fetch failed: ${err.message}`);
  }
}

/**
 */
  const { maxResults = 40 } = options;


  const ZILLOW_ACTOR_ID = 'maxcopell~zillow-zip-search';

  try {
    // Start the actor run
    const runResponse = await axios.post(
      {
        zipCodes: [location],
        maxItems: maxResults
      },
      { timeout: 10000 }
    );

    const runId = runResponse.data?.data?.id;


    // Poll for completion (Zillow scrape takes ~30-90 seconds)
    let attempts = 0;
    const maxAttempts = 20;

    while (attempts < maxAttempts) {
      await new Promise(r => setTimeout(r, 5000)); // wait 5s
      attempts++;

      const statusResponse = await axios.get(
        { timeout: 10000 }
      );

      const status = statusResponse.data?.data?.status;
      process.stderr.write(`\r  ⏳ Status: ${status} (${attempts * 5}s)...`);

      if (status === 'SUCCEEDED') {
        break;
      }
      if (status === 'FAILED' || status === 'ABORTED') {
      }
    }

    // Fetch results from dataset
    const datasetResponse = await axios.get(
      { timeout: 30000 }
    );

    const items = datasetResponse.data || [];


  } catch (err) {
  }
}

/**
 * Normalize a RapidAPI Zillow result to our standard format
 */
function normalizeZillowListing(item) {
  // Parse city/state/zip from address string if not separate fields
  // Format: "1112 W Annie St, Austin, TX 78704"
  let city = item.city || '';
  let state = item.state || '';
  let zip = item.zipcode || item.zip || '';
  
  if (!city && item.address) {
    const parts = item.address.split(',').map(s => s.trim());
    if (parts.length >= 2) {
      city = parts[parts.length - 2] || '';
      const stateZip = (parts[parts.length - 1] || '').split(' ');
      state = stateZip[0] || '';
      zip = stateZip[1] || zip;
    }
  }

  return {
    zpid: item.zpid,
    address: item.address,
    city,
    state,
    zip,
    price: item.price,
    beds: item.bedrooms,
    baths: item.bathrooms,
    sqft: item.livingArea,
    yearBuilt: item.yearBuilt,
    propertyType: item.homeType || item.propertyType || 'Residential',
    daysOnMarket: item.daysOnZillow || item.daysOnMarket,
    zestimate: item.zestimate,
    listingUrl: item.detailUrl ? `https://zillow.com${item.detailUrl}` : `https://zillow.com/homes/${item.zpid}_zpid`,
    lat: item.latitude,
    lng: item.longitude,
    pricePerSqft: item.priceReduction ? null : Math.round(item.price / item.livingArea)
  };
}

/**
 */
  // Handle maxcopell~zillow-zip-search output format
  // Top-level: beds, baths, area, price (string "$824,000"), addressStreet, detailUrl, zpid
  // hdpData.homeInfo: bedrooms, bathrooms, livingArea, homeType, daysOnZillow, latitude, longitude, streetAddress
  const hdp = item.hdpData?.homeInfo || {};
  const addr = item.addressStreet || hdp.streetAddress || (typeof item.address === 'string' ? item.address : item.address?.streetAddress) || '';
  const city = item.addressCity || hdp.city || '';
  const state = item.addressState || hdp.state || '';
  const zip = item.addressZipcode || hdp.zipcode || '';
  const fullAddr = addr.includes(',') ? addr : `${addr}, ${city}, ${state} ${zip}`.trim();

  // Price: prefer numeric, fall back to parsing string
  let price = item.unformattedPrice || hdp.price;
  if (!price && typeof item.price === 'string') {
    price = parseInt(item.price.replace(/[^0-9]/g, ''), 10) || 0;
  }
  if (!price) price = item.price;

  const beds = item.beds || hdp.bedrooms || item.bedrooms;
  const baths = item.baths || hdp.bathrooms || item.bathrooms;
  const sqft = item.area || hdp.livingArea || item.livingArea;

  return {
    zpid: item.zpid || item.id,
    address: fullAddr,
    city, state, zip,
    price,
    beds,
    baths,
    sqft,
    yearBuilt: item.yearBuilt || hdp.yearBuilt,
    propertyType: hdp.homeType || item.homeType || item.statusText,
    daysOnMarket: hdp.daysOnZillow || item.daysOnMarket || item.daysOnZillow,
    zestimate: item.zestimate || hdp.zestimate,
    listingUrl: item.detailUrl || item.url || `https://zillow.com/homes/${item.zpid || item.id}_zpid`,
    lat: hdp.latitude || item.latLong?.latitude || item.latitude,
    lng: hdp.longitude || item.latLong?.longitude || item.longitude,
    pricePerSqft: price && sqft ? Math.round(price / sqft) : null
  };
}

module.exports = { fetchZillowListings };
