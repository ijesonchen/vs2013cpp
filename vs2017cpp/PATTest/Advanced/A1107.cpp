/*

*/

#include "..\patMain.h"
#include <iostream>
#include <string>

using namespace std;

// rename this to main int PAT
int A1107Func(void)
{
	return 0;
}


void A1107(const string& fn)
{
	cout << fn << endl;
	RedirCin(fn);
	A1107Func();
	cout << endl;
}

void A1107(void)
{
	A1107("data\\A1107-1.txt"); // 
}

