package dude.morrildl.providence.util;

import java.io.BufferedInputStream;
import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.FileInputStream;
import java.io.FileNotFoundException;
import java.io.IOException;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.MalformedURLException;
import java.net.URL;
import java.security.KeyManagementException;
import java.security.KeyStore;
import java.security.KeyStoreException;
import java.security.NoSuchAlgorithmException;
import java.security.cert.CertificateException;

import javax.net.ssl.SSLContext;
import javax.net.ssl.SSLSocketFactory;
import javax.net.ssl.TrustManagerFactory;

import android.content.Context;
import android.content.res.Resources.NotFoundException;
import android.net.Uri;
import android.os.ParcelFileDescriptor;
import android.util.Log;
import dude.morrildl.providence.R;

public class Network {
	private static Network instance = null;

	public static Network getInstance(Context context) throws KeyStoreException {
		if (instance == null) {
			synchronized (Network.class) {
				if (instance == null) {
					instance = new Network(context);
				}
			}
		}
		if (instance.sslContext == null) {
			throw new KeyStoreException("failure preparing SSL keystore");
		}
		return instance;
	}

	private Context context = null;

	private SSLContext sslContext = null;

	private Network(Context context) {
		this.context = context;
		try {
			KeyStore ks = KeyStore.getInstance("BKS");
			ks.load(context.getResources().openRawResource(R.raw.keystore),
					"boogaflex".toCharArray());
			TrustManagerFactory tmf = TrustManagerFactory
					.getInstance(TrustManagerFactory.getDefaultAlgorithm());
			tmf.init(ks);
			sslContext = SSLContext.getInstance("TLS");
			sslContext.init(null, tmf.getTrustManagers(), null);
		} catch (KeyStoreException e) {
		} catch (KeyManagementException e) {
		} catch (NoSuchAlgorithmException e) {
		} catch (CertificateException e) {
		} catch (NotFoundException e) {
		} catch (IOException e) {
		}
	}

	public SSLSocketFactory getSslSocketFactory() {
		return sslContext.getSocketFactory();
	}

	public byte[] readFile(File f) {
		int length = (int) f.length();
		byte[] bytes = null;
		try {
			bytes = readStream(length, new FileInputStream(f));
		} catch (FileNotFoundException e) {
			Log.w("Network.readFile",
					"called to read nonexistent file " + f.getPath());
			return null;
		} catch (IOException e) {
			Log.w("Network.readFile", "failed during read", e);
			return null;
		}
		return bytes;
	}

	public byte[] readProvider(String url) {
		try {
			ParcelFileDescriptor pfd = context.getContentResolver()
					.openFileDescriptor(Uri.parse(url), "r");
			FileInputStream fis = new FileInputStream(pfd.getFileDescriptor());
			if (pfd.getStatSize() > 0) {
				return readStream((int) pfd.getStatSize(), fis);
			}

			ByteArrayOutputStream baos = new ByteArrayOutputStream();
			byte[] buf = new byte[4096];
			while (true) {
				int n = fis.read(buf);
				if (n < 0)
					break;
				baos.write(buf, 0, n);
			}
			return baos.toByteArray();
		} catch (FileNotFoundException e) {
			Log.w("Network.readProvider", "file not found?", e);
		} catch (IOException e) {
			Log.w("Network.readProvider", "IOException during read", e);
		}

		return null;
	}

	public byte[] readStream(int length, InputStream is) throws IOException {
		byte[] bytes = new byte[length];
		BufferedInputStream bir = new BufferedInputStream(is);
		int num = 0;
		while (num < length) {
			num += bir.read(bytes, num, length - num);
		}
		return bytes;
	}

	public byte[] readUrl(String url) {
		try {
			URL uri = new URL(url);
			HttpURLConnection cxn = (HttpURLConnection) uri.openConnection();
			int length = cxn.getContentLength();
			return readStream(length, cxn.getInputStream());
		} catch (MalformedURLException e) {
			Log.w("Network.readUrl", "called with hosed URL " + url);
			return null;
		} catch (IOException e) {
			Log.w("Network.readUrl", "IOException during read", e);
			return null;
		}
	}

	public URL urlForResource(int resource, String query)
			throws MalformedURLException {
		InputStream is = context.getResources().openRawResource(resource);
		String s = new java.util.Scanner(is).useDelimiter("\\A").next().trim();
		if (query != null) {
			s += query;
		}
		return new URL(s);
	}
}
