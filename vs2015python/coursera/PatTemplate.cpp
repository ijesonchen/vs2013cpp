/*

*/

#include "patMain.h"
#include <iostream>
#include <string>

using namespace std;

// rename this to main int PAT
int A0000Func(void)
{
	return 0;
}


void A0000(const string& fn)
{
	cout << fn << endl;
	RedirCin(fn);
	A0000Func();
	cout << endl;
}

void A0000(void)
{
	A0000("data\\A0000-1.txt"); // 
}

